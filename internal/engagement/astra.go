package engagement

import (
	"context"
	"errors"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	gocqlastra "github.com/datastax/gocql-astra/v2"

	"litloop/backend/internal/config"
)

const schema = `
CREATE TABLE IF NOT EXISTS events_by_resource_day (
    resource_type text,
    resource_id bigint,
    bucket_day text,
    occurred_at timestamp,
    event_id bigint,
    actor_id bigint,
    action text,
    PRIMARY KEY ((resource_type, resource_id, bucket_day), occurred_at, event_id)
) WITH CLUSTERING ORDER BY (occurred_at DESC, event_id DESC)`

type Store struct {
	session *gocql.Session
}

func Open(ctx context.Context, c config.Config) (*Store, error) {
	cluster, err := gocqlastra.NewClusterFromURL("https://api.astra.datastax.com", c.AstraDatabaseID, c.AstraToken, 5*time.Second)
	if err != nil {
		return nil, errors.New("could not load the Astra DB connection bundle")
	}
	cluster.Keyspace = c.AstraKeyspace
	cluster.Timeout = 5 * time.Second
	cluster.ConnectTimeout = 5 * time.Second
	cluster.Consistency = gocql.LocalQuorum
	session, err := gocql.NewSession(*cluster)
	if err != nil {
		return nil, errors.New("could not connect to the Astra DB keyspace")
	}
	if err := session.Query(schema).WithContext(ctx).Exec(); err != nil {
		session.Close()
		return nil, errors.New("could not initialize the Cassandra engagement table")
	}
	return &Store{session: session}, nil
}

func (s *Store) Close() {
	s.session.Close()
}

func (s *Store) Record(ctx context.Context, id int64, event Event) error {
	at := event.OccurredAt.UTC()
	// The task ID and time stay the same on retry, so INSERT overwrites the same row.
	return s.session.Query(`INSERT INTO events_by_resource_day
		(resource_type, resource_id, bucket_day, occurred_at, event_id, actor_id, action)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ResourceType, event.ResourceID, at.Format(time.DateOnly), at, id, event.ActorID, event.Action).
		WithContext(ctx).Exec()
}
