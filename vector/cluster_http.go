package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPTransport implements Transport over HTTP. Peer addresses may be either a
// bare host:port or a full base URL. The transport posts JSON payloads to the
// embedded vector cluster endpoints exposed by every node.
type HTTPTransport struct {
	client *http.Client
	token  string
}

// NewHTTPTransport creates a new HTTP-based cluster transport. The optional
// token is sent as an Authorization bearer header so internal cluster traffic
// can be authenticated.
func NewHTTPTransport(client *http.Client, token string) *HTTPTransport {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPTransport{client: client, token: token}
}

// SendHeartbeat posts a heartbeat to the peer and decodes the reply.
func (t *HTTPTransport) SendHeartbeat(ctx context.Context, peer string, hb Heartbeat) (HeartbeatReply, error) {
	var reply HeartbeatReply
	if err := t.post(ctx, peer, "/api/vector/cluster/heartbeat", hb, &reply); err != nil {
		return HeartbeatReply{}, err
	}
	return reply, nil
}

// Replicate pushes an operation to the peer follower.
func (t *HTTPTransport) Replicate(ctx context.Context, peer string, op ReplicatedOperation) error {
	return t.post(ctx, peer, "/api/vector/cluster/replicate", op, nil)
}

// Forward proxies an operation to the cluster leader.
func (t *HTTPTransport) Forward(ctx context.Context, peer string, op ReplicatedOperation) error {
	return t.post(ctx, peer, "/api/vector/cluster/forward", op, nil)
}

// SendSnapshot posts a database snapshot file to the peer follower.
func (t *HTTPTransport) SendSnapshot(ctx context.Context, peer string, snapshotReader io.Reader, lastLogIndex uint64, appliedLogIndex uint64, term uint64) error {
	url := normalizePeerURL(peer) + fmt.Sprintf("/api/vector/cluster/install-snapshot?lastLogIndex=%d&appliedLogIndex=%d&term=%d", lastLogIndex, appliedLogIndex, term)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, snapshotReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("cluster peer %s install-snapshot returned %d: %s", peer, resp.StatusCode, string(data))
	}
	return nil
}

// Join posts a join request to the cluster leader to dynamically register selfAddr.
func (t *HTTPTransport) Join(ctx context.Context, leader string, selfAddr string) error {
	payload := map[string]string{"addr": selfAddr}
	return t.post(ctx, leader, "/api/vector/cluster/join", payload, nil)
}

func (t *HTTPTransport) post(ctx context.Context, peer, path string, payload, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := normalizePeerURL(peer) + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("cluster peer %s returned %d: %s", peer, resp.StatusCode, string(data))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func normalizePeerURL(peer string) string {
	peer = strings.TrimRight(strings.TrimSpace(peer), "/")
	if peer == "" {
		return peer
	}
	if strings.HasPrefix(peer, "http://") || strings.HasPrefix(peer, "https://") {
		return peer
	}
	return "http://" + peer
}
