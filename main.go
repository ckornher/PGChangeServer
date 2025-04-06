package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	pb "PGChangeServer/proto"
	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"google.golang.org/grpc"
)

const (
	defaultPageSize = 1000
	slotName        = "my_replication_slot"
	publicationName = "mypub"
)

type server struct {
	pb.UnimplementedDatabaseChangeStreamServer
	mu        sync.Mutex
	conn      *pgx.Conn
	eventChan chan *pb.DatabaseEvent
	stopChan  chan struct{}
	isRunning bool
	pageSize  int
	tables    []string
	relations map[uint32]*pglogrepl.RelationMessage // Cache for relation metadata
}

func NewServer() *server {
	return &server{
		eventChan: make(chan *pb.DatabaseEvent, 1000),
		stopChan:  make(chan struct{}),
		relations: make(map[uint32]*pglogrepl.RelationMessage),
	}
}

func (s *server) log(msg string, args ...interface{}) {
	fmt.Printf("[%s] [INFO] %s\n", time.Now().UTC().Format(time.RFC3339), fmt.Sprintf(msg, args...))
}

func (s *server) logError(msg string, err error) {
	fmt.Printf("[%s] [ERROR] %s: %v\n", time.Now().UTC().Format(time.RFC3339), msg, err)
}

func (s *server) sendEvent(event *pb.DatabaseEvent) {
	select {
	case s.eventChan <- event:
	case <-s.stopChan:
	}
}

func (s *server) StartStream(req *pb.StreamRequest, stream pb.DatabaseChangeStream_StartStreamServer) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("stream already running")
	}
	s.isRunning = true
	s.pageSize = int(req.PageSize)
	if s.pageSize <= 0 {
		s.pageSize = defaultPageSize
	}
	s.tables = req.Tables
	s.mu.Unlock()

	ctx := stream.Context()
	s.sendEvent(&pb.DatabaseEvent{
		EventType: &pb.DatabaseEvent_Status{
			Status: &pb.StreamStatus{State: "INITIALIZING", Message: "Starting stream", Timestamp: time.Now().UnixMilli()},
		},
	})

	if err := s.setupReplication(ctx, req); err != nil {
		s.sendEvent(&pb.DatabaseEvent{
			EventType: &pb.DatabaseEvent_Status{
				Status: &pb.StreamStatus{State: "FAILED", Message: fmt.Sprintf("Setup failed: %v", err), Timestamp: time.Now().UnixMilli()},
			},
		})
		s.cleanup()
		return err
	}

	s.streamEvents(ctx, stream)
	return nil
}

func (s *server) StopStream(ctx context.Context, _ *pb.StopRequest) (*pb.StopResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isRunning {
		return &pb.StopResponse{Message: "Stream not running"}, nil
	}
	s.log("Stopping stream")
	close(s.stopChan)
	s.cleanup()
	s.sendEvent(&pb.DatabaseEvent{
		EventType: &pb.DatabaseEvent_Status{
			Status: &pb.StreamStatus{State: "STOPPED", Message: "Stream stopped", Timestamp: time.Now().UnixMilli()},
		},
	})
	return &pb.StopResponse{Message: "Stream stopped"}, nil
}

func (s *server) SyncStream(req *pb.StreamRequest, stream pb.DatabaseChangeStream_SyncStreamServer) error {
	s.mu.Lock()
	if s.isRunning {
		s.cleanup()
	}
	s.isRunning = true
	s.pageSize = int(req.PageSize)
	if s.pageSize <= 0 {
		s.pageSize = defaultPageSize
	}
	s.tables = req.Tables
	s.mu.Unlock()

	ctx := stream.Context()
	s.sendEvent(&pb.DatabaseEvent{
		EventType: &pb.DatabaseEvent_Status{
			Status: &pb.StreamStatus{State: "INITIALIZING", Message: "Syncing stream", Timestamp: time.Now().UnixMilli()},
		},
	})

	if err := s.setupReplication(ctx, req); err != nil {
		s.sendEvent(&pb.DatabaseEvent{
			EventType: &pb.DatabaseEvent_Status{
				Status: &pb.StreamStatus{State: "FAILED", Message: fmt.Sprintf("Setup failed: %v", err), Timestamp: time.Now().UnixMilli()},
			},
		})
		s.cleanup()
		return err
	}

	s.streamEvents(ctx, stream)
	return nil
}

func (s *server) setupReplication(ctx context.Context, req *pb.StreamRequest) error {
	connString := "postgres://user:password@localhost:5432/your_db?replication=database" // Update credentials
	config, err := pgx.ParseConfig(connString)
	if err != nil {
		return err
	}

	s.conn, err = pgx.ConnectConfig(ctx, config)
	if err != nil {
		return err
	}

	// Drop slot if starting fresh
	if req.LastKnownTransactionId == 0 {
		_, err = s.conn.Exec(ctx, fmt.Sprintf("DROP_REPLICATION_SLOT %s", slotName))
		if err != nil && !isDuplicateSlot(err) {
			s.logError("Drop slot failed (ignoring if not exists)", err)
		}
	}

	// Create replication slot
	slotOptions := pglogrepl.CreateReplicationSlotOptions{
		Temporary: false,
		Mode:      pglogrepl.LogicalReplication,
	}
	resp, err := pglogrepl.CreateReplicationSlot(ctx, s.conn.PgConn(), slotName, "pgoutput", slotOptions)
	if err != nil && !isDuplicateSlot(err) {
		return err
	}
	lsn := resp.ConsistentPoint

	if req.IncludeSnapshot {
		if err := s.sendSnapshot(ctx); err != nil {
			return err
		}
		s.sendEvent(&pb.DatabaseEvent{
			EventType: &pb.DatabaseEvent_SnapshotEnd{
				SnapshotEnd: &pb.SnapshotEnd{Timestamp: time.Now().UnixMilli(), Lsn: lsn},
			},
		})
	}

	go s.processReplication(ctx, s.conn.PgConn())
	return nil
}

func (s *server) sendSnapshot(ctx context.Context) error {
	s.sendEvent(&pb.DatabaseEvent{
		EventType: &pb.DatabaseEvent_SnapshotBegin{
			SnapshotBegin: &pb.SnapshotBegin{Tables: s.tables, Timestamp: time.Now().UnixMilli(), PageSize: int32(s.pageSize)},
		},
	})

	rows, err := s.conn.Query(ctx, "SELECT id, name, updated_at FROM your_table_name") // Update table name
	if err != nil {
		s.logError("Snapshot query failed", err)
		s.sendEvent(&pb.DatabaseEvent{
			EventType: &pb.DatabaseEvent_SnapshotFailure{
				SnapshotFailure: &pb.SnapshotFailure{ErrorMessage: err.Error(), Timestamp: time.Now().UnixMilli()},
			},
		})
		return err
	}
	defer rows.Close()

	pageNum := 0
	rowCount := 0
	for rows.Next() {
		if rowCount%s.pageSize == 0 {
			if rowCount > 0 {
				s.sendEvent(&pb.DatabaseEvent{
					EventType: &pb.DatabaseEvent_PageEnd{
						PageEnd: &pb.SnapshotPageEnd{PageNumber: int32(pageNum)},
					},
				})
				pageNum++
			}
			s.sendEvent(&pb.DatabaseEvent{
				EventType: &pb.DatabaseEvent_PageBegin{
					PageBegin: &pb.SnapshotPageBegin{PageNumber: int32(pageNum), RowCount: int32(min(s.pageSize, s.pageSize))},
				},
			})
		}

		var id int
		var name string
		var updatedAt time.Time
		if err := rows.Scan(&id, &name, &updatedAt); err != nil {
			s.logError("Snapshot row scan failed", err)
			s.sendEvent(&pb.DatabaseEvent{
				EventType: &pb.DatabaseEvent_SnapshotFailure{
					SnapshotFailure: &pb.SnapshotFailure{ErrorMessage: err.Error(), Timestamp: time.Now().UnixMilli()},
				},
			})
			return err
		}

		values := map[string]string{
			"id":         fmt.Sprintf("%d", id),
			"name":       name,
			"updated_at": updatedAt.String(),
		}
		s.sendEvent(&pb.DatabaseEvent{
			EventType: &pb.DatabaseEvent_SnapshotRow{
				SnapshotRow: &pb.SnapshotRow{TableName: "your_table_name", Values: values},
			},
		})
		rowCount++
	}

	if rowCount > 0 {
		s.sendEvent(&pb.DatabaseEvent{
			EventType: &pb.DatabaseEvent_PageEnd{
				PageEnd: &pb.SnapshotPageEnd{PageNumber: int32(pageNum)},
			},
		})
	}
	return nil
}

func (s *server) processReplication(ctx context.Context, conn *pgconn.PgConn) {
	s.sendEvent(&pb.DatabaseEvent{
		EventType: &pb.DatabaseEvent_Status{
			Status: &pb.StreamStatus{State: "STREAMING", Message: "Replication started", Timestamp: time.Now().UnixMilli()},
		},
	})

	err := pglogrepl.StartReplication(ctx, conn, slotName, 0, pglogrepl.StartReplicationOptions{
		PluginArgs: []string{"proto_version '1'", fmt.Sprintf("publication_names '%s'", publicationName)},
	})
	if err != nil {
		s.logError("Start replication failed", err)
		return
	}

	var currentXid uint32
	var lastLSN pglogrepl.LSN
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		default:
			msg, err := conn.ReceiveMessage(ctx)
			if err != nil {
				s.logError("Receive message failed", err)
				return
			}
			if msg == nil {
				continue
			}

			switch m := msg.(type) {
			case *pgproto3.CopyData:
				switch m.Data[0] {
				case pglogrepl.XLogDataByteID:
					xld, err := pglogrepl.ParseXLogData(m.Data[1:])
					if err != nil {
						s.logError("Parse XLogData failed", err)
						continue
					}
					lm, err := pglogrepl.Parse(xld.WALData)
					if err != nil {
						s.logError("Parse logical message failed", err)
						continue
					}

					switch lm := lm.(type) {
					case *pglogrepl.BeginMessage:
						currentXid = lm.Xid
						s.sendEvent(&pb.DatabaseEvent{
							EventType: &pb.DatabaseEvent_Begin{
								Begin: &pb.TransactionBegin{
									TransactionId: lm.Xid,
									Timestamp:     time.Now().UnixMilli(),
									Lsn:           lm.FinalLSN.String(),
								},
							},
						})
					case *pglogrepl.RelationMessage:
						s.mu.Lock()
						s.relations[lm.RelationID] = lm
						s.mu.Unlock()
					case *pglogrepl.InsertMessage:
						rel, ok := s.relations[lm.RelationID]
						if !ok {
							s.log("Unknown relation ID: %d", lm.RelationID)
							continue
						}
						values := map[string]string{}
						for i, col := range rel.Columns {
							if i < len(lm.Tuple.Columns) && lm.Tuple.Columns[i].Data != nil {
								values[col.Name] = string(lm.Tuple.Columns[i].Data)
							}
						}
						s.sendEvent(&pb.DatabaseEvent{
							EventType: &pb.DatabaseEvent_Change{
								Change: &pb.ChangeEvent{
									TransactionId: currentXid,
									TableName:     rel.RelationName,
									Operation:     "INSERT",
									Timestamp:     time.Now().UnixMilli(),
									NewValues:     values,
								},
							},
						})
					case *pglogrepl.UpdateMessage:
						rel, ok := s.relations[lm.RelationID]
						if !ok {
							s.log("Unknown relation ID: %d", lm.RelationID)
							continue
						}
						oldValues := map[string]string{}
						newValues := map[string]string{}
						for i, col := range rel.Columns {
							if lm.OldTuple != nil && i < len(lm.OldTuple.Columns) && lm.OldTuple.Columns[i].Data != nil {
								oldValues[col.Name] = string(lm.OldTuple.Columns[i].Data)
							}
							if lm.NewTuple != nil && i < len(lm.NewTuple.Columns) && lm.NewTuple.Columns[i].Data != nil {
								newValues[col.Name] = string(lm.NewTuple.Columns[i].Data)
							}
						}
						s.sendEvent(&pb.DatabaseEvent{
							EventType: &pb.DatabaseEvent_Change{
								Change: &pb.ChangeEvent{
									TransactionId: currentXid,
									TableName:     rel.RelationName,
									Operation:     "UPDATE",
									Timestamp:     time.Now().UnixMilli(),
									OldValues:     oldValues,
									NewValues:     newValues,
								},
							},
						})
					case *pglogrepl.DeleteMessage:
						rel, ok := s.relations[lm.RelationID]
						if !ok {
							s.log("Unknown relation ID: %d", lm.RelationID)
							continue
						}
						values := map[string]string{}
						for i, col := range rel.Columns {
							if i < len(lm.OldTuple.Columns) && lm.OldTuple.Columns[i].Data != nil {
								values[col.Name] = string(lm.OldTuple.Columns[i].Data)
							}
						}
						s.sendEvent(&pb.DatabaseEvent{
							EventType: &pb.DatabaseEvent_Change{
								Change: &pb.ChangeEvent{
									TransactionId: currentXid,
									TableName:     rel.RelationName,
									Operation:     "DELETE",
									Timestamp:     time.Now().UnixMilli(),
									OldValues:     values,
								},
							},
						})
					case *pglogrepl.CommitMessage:
						s.sendEvent(&pb.DatabaseEvent{
							EventType: &pb.DatabaseEvent_Commit{
								Commit: &pb.TransactionCommit{
									TransactionId:   currentXid,
									CommitTimestamp: lm.CommitTime.UnixMilli(),
									Lsn:             lm.CommitLSN.String(),
								},
							},
						})
					}
					lastLSN = xld.WALStart
				case pglogrepl.PrimaryKeepaliveMessageByteID:
					pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(m.Data[1:])
					if err != nil {
						s.logError("Parse keepalive failed", err)
						continue
					}
					if pkm.ReplyRequested {
						err = pglogrepl.SendStandbyStatusUpdate(ctx, conn, pglogrepl.StandbyStatusUpdate{WALWritePosition: lastLSN})
						if err != nil {
							s.logError("Send standby status failed", err)
							return
						}
					}
				}
			}

			// Send standby status update periodically or on demand
			if lastLSN != 0 {
				err = pglogrepl.SendStandbyStatusUpdate(ctx, conn, pglogrepl.StandbyStatusUpdate{WALWritePosition: lastLSN})
				if err != nil {
					s.logError("Send standby status failed", err)
					return
				}
			}
		}
	}
}

func (s *server) streamEvents(ctx context.Context, stream grpc.ServerStream) {
	for {
		select {
		case <-ctx.Done():
			s.cleanup()
			return
		case <-s.stopChan:
			s.cleanup()
			return
		case event := <-s.eventChan:
			if err := stream.SendMsg(event); err != nil {
				s.logError("Stream send failed", err)
				s.cleanup()
				return
			}
		}
	}
}

func (s *server) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		s.conn.Close(context.Background())
		s.conn = nil
	}
	s.isRunning = false
	s.stopChan = make(chan struct{})
	s.relations = make(map[uint32]*pglogrepl.RelationMessage)
}

func isDuplicateSlot(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "42710" // duplicate_object
	}
	return false
}

//func min(a, b int) int {
//	if a < b {
//		return a
//	}
//	return b
//}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}
	s := grpc.NewServer()
	pb.RegisterDatabaseChangeStreamServer(s, NewServer())
	fmt.Println("Server listening on :50051")
	if err := s.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}
