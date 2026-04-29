package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"
	"gocloud.dev/blob"
	"gocloud.dev/blob/driver"
	"gocloud.dev/gcerrors"
)

type webdavBucket struct {
	client *gowebdav.Client
}

func (b *webdavBucket) Close() error {
	return nil
}

func (b *webdavBucket) ErrorCode(err error) gcerrors.ErrorCode {
	if os.IsNotExist(err) || (err != nil && strings.Contains(err.Error(), "404")) {
		return gcerrors.NotFound
	}
	return gcerrors.Unknown
}

func (b *webdavBucket) ErrorAs(err error, i interface{}) bool {
	return false
}

func (b *webdavBucket) Attributes(ctx context.Context, key string) (*driver.Attributes, error) {
	info, err := b.client.Stat(key)
	if err != nil {
		return nil, err
	}

	return &driver.Attributes{
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

func (b *webdavBucket) ListPaged(ctx context.Context, opts *driver.ListOptions) (*driver.ListPage, error) {
	// WebDAV listing is typically not paged in gowebdav
	// This is a simple implementation
	files, err := b.client.ReadDir(opts.Prefix)
	if err != nil {
		// If the prefix itself is a file, or doesn't exist
		return nil, err
	}

	var objects []*driver.ListObject
	for _, f := range files {
		key := path.Join(opts.Prefix, f.Name())
		if f.IsDir() {
			key += "/"
		}
		objects = append(objects, &driver.ListObject{
			Key:     key,
			ModTime: f.ModTime(),
			Size:    f.Size(),
			IsDir:   f.IsDir(),
		})
	}

	return &driver.ListPage{Objects: objects}, nil
}

func (b *webdavBucket) NewRangeReader(ctx context.Context, key string, offset, length int64, opts *driver.ReaderOptions) (driver.Reader, error) {
	// Simple implementation: download everything if range is not easily supported by gowebdav
	// But gowebdav doesn't seem to support range requests easily.
	// For now, let's just get the whole file.
	
	reader, err := b.client.ReadStream(key)
	if err != nil {
		return nil, err
	}

	// Handle offset
	if offset > 0 {
		if _, err := io.CopyN(io.Discard, reader, offset); err != nil {
			return nil, err
		}
	}

	var rc io.ReadCloser = reader
	if length >= 0 {
		rc = struct {
			io.Reader
			io.Closer
		}{
			Reader: io.LimitReader(reader, length),
			Closer: reader,
		}
	}

	info, _ := b.client.Stat(key)
	
	return &webdavReader{
		rc:      rc,
		size:    info.Size(),
		modTime: info.ModTime(),
	}, nil
}

type webdavReader struct {
	rc      io.ReadCloser
	size    int64
	modTime time.Time
}

func (r *webdavReader) Read(p []byte) (int, error) {
	return r.rc.Read(p)
}

func (r *webdavReader) Close() error {
	return r.rc.Close()
}

func (r *webdavReader) Attributes() *driver.ReaderAttributes {
	return &driver.ReaderAttributes{
		Size:    r.size,
		ModTime: r.modTime,
	}
}

func (b *webdavBucket) NewTypedWriter(ctx context.Context, key string, contentType string, opts *driver.WriterOptions) (driver.Writer, error) {
	// Ensure parent directories exist
	dir := path.Dir(key)
	if dir != "." && dir != "/" {
		parts := strings.Split(strings.Trim(dir, "/"), "/")
		current := ""
		for _, part := range parts {
			current = path.Join(current, part)
			b.client.Mkdir(current, 0755)
		}
	}

	pr, pw := io.Pipe()

	go func() {
		err := b.client.WriteStream(key, pr, 0644)
		if err != nil {
			pr.CloseWithError(err)
		} else {
			pr.Close()
		}
	}()

	return &webdavWriter{
		pw: pw,
	}, nil
}

type webdavWriter struct {
	pw *io.PipeWriter
}

func (w *webdavWriter) Write(p []byte) (int, error) {
	return w.pw.Write(p)
}

func (w *webdavWriter) Close() error {
	return w.pw.Close()
}

func (b *webdavBucket) Copy(ctx context.Context, dstKey, srcKey string, opts *driver.CopyOptions) error {
	return b.client.Copy(srcKey, dstKey, true)
}

func (b *webdavBucket) Delete(ctx context.Context, key string) error {
	return b.client.Remove(key)
}

func (b *webdavBucket) SignedURL(ctx context.Context, key string, opts *driver.SignedURLOptions) (string, error) {
	return "", fmt.Errorf("SignedURL not supported for WebDAV")
}

// NewWebDAV initializes a WebDAV filesystem instance.
func NewWebDAV(url, username, password string) (*System, error) {
	client := gowebdav.NewClient(url, username, password)
	
	// Test connection
	if err := client.Connect(); err != nil {
		return nil, err
	}

	bucket := blob.NewBucket(&webdavBucket{client: client})
	
	return &System{
		ctx:    context.Background(),
		bucket: bucket,
	}, nil
}

func (b *webdavBucket) As(i interface{}) bool { return false }
func (r *webdavReader) As(i interface{}) bool { return false }
func (w *webdavWriter) As(i interface{}) bool { return false }
