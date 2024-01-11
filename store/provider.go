package store

import (
	"io"
	"os"
	"time"

	"github.com/rqlite/rqlite/v8/command/proto"
)

// Provider implements the uploader Provider interface, allowing the
// Store to be used as a DataProvider for an uploader.
type Provider struct {
	str    *Store
	vacuum bool

	nRetries      int
	retryInterval time.Duration
}

// NewProvider returns a new instance of Provider.
func NewProvider(s *Store, v bool) *Provider {
	return &Provider{
		str:           s,
		vacuum:        v,
		nRetries:      10,
		retryInterval: 500 * time.Millisecond,
	}
}

// LastModified returns the time the data managed by the Provider was
// last modified.
func (p *Provider) LastModified() (time.Time, error) {
	stats.Add(numProviderChecks, 1)
	return p.str.db.LastModified()
}

// Provider writes the SQLite database to the given path. If path exists,
// it will be overwritten.
func (p *Provider) Provide(path string) (t time.Time, retErr error) {
	stats.Add(numProviderProvides, 1)
	defer func() {
		if retErr != nil {
			stats.Add(numProviderProvidesFail, 1)
		}
	}()

	fd, err := os.Create(path)
	if err != nil {
		return time.Time{}, err
	}
	defer fd.Close()
	return p.provide(fd)
}

func (p *Provider) provide(w io.Writer) (time.Time, error) {
	br := &proto.BackupRequest{
		Format: proto.BackupRequest_BACKUP_REQUEST_FORMAT_BINARY,
		Vacuum: p.vacuum,
	}
	nRetries := 0
	for {
		err := p.str.Backup(br, w)
		if err == nil {
			break
		}
		time.Sleep(p.retryInterval)
		nRetries++
		if nRetries > p.nRetries {
			return time.Time{}, err
		}
	}
	return p.str.db.LastModified()
}
