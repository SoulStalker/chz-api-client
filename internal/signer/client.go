package signer

import (
	"context"

	"github.com/SoulStalker/chz-api-client/internal/model"
	pb "github.com/SoulStalker/sign-service/gen/signer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	signer pb.SignerClient
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, signer: pb.NewSignerClient(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ListCertificates(ctx context.Context) ([]model.Certificate, error) {
	resp, err := c.signer.ListCertificates(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	certs := make([]model.Certificate, 0, len(resp.Certificates))
	for _, cert := range resp.Certificates {
		certs = append(certs, model.Certificate{
			Thumbprint: cert.Thumbprint,
			Subject:    cert.Subject,
			Issuer:     cert.Issuer,
			NotAfter:   cert.NotAfter,
		})
	}
	return certs, nil
}

func (c *Client) Sign(ctx context.Context, data []byte, thumbprint, callerID string) ([]byte, error) {
	resp, err := c.signer.Sign(ctx, &pb.SignRequest{
		Payload:    data,
		Thumbprint: thumbprint,
		CallerId:   callerID,
	})
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}
