package client

import (
	"context"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/lager"

	coreosetcdclient "github.com/coreos/etcd/client"
	"github.com/coreos/etcd/pkg/transport"
)

type EtcdClient struct {
	coreosEtcdClient coreosetcdclient.Client
	membersAPI       coreosetcdclient.MembersAPI

	logger logger
}

type Member struct {
	ID         string
	Name       string
	PeerURLs   []string
	ClientURLs []string
}

type Config interface {
	EtcdClientEndpoints() []string
	RequireSSL() bool
}

type logger interface {
	Info(string, ...lager.Data)
	Error(string, error, ...lager.Data)
}

func NewEtcdClient(logger logger) *EtcdClient {
	return &EtcdClient{
		logger: logger,
	}
}

func (e *EtcdClient) Configure(etcdfabConfig Config, certDir string) error {
	endpoints := etcdfabConfig.EtcdClientEndpoints()
	e.logger.Info("etcd-client.configure.config", lager.Data{
		"endpoints": endpoints,
	})

	tns := coreosetcdclient.DefaultTransport

	if etcdfabConfig.RequireSSL() {
		caCertFile := filepath.Join(certDir, "server-ca.crt")
		clientCertFile := filepath.Join(certDir, "client.crt")
		clientKeyFile := filepath.Join(certDir, "client.key")

		tlsInfo := transport.TLSInfo{
			CAFile:         caCertFile,
			CertFile:       clientCertFile,
			KeyFile:        clientKeyFile,
			ClientCertAuth: etcdfabConfig.RequireSSL(),
		}

		var err error
		tns, err = transport.NewTransport(tlsInfo)
		if err != nil {
			panic(err)
			// return err
		}
	}

	cfg := coreosetcdclient.Config{
		Endpoints:               endpoints,
		Transport:               tns,
		HeaderTimeoutPerRequest: time.Second,
	}
	coreosEtcdClient, err := coreosetcdclient.New(cfg)
	if err != nil {
		return err
	}

	membersAPI := coreosetcdclient.NewMembersAPI(coreosEtcdClient)

	e.coreosEtcdClient = coreosEtcdClient
	e.membersAPI = membersAPI

	return nil
}

func (e *EtcdClient) MemberList() ([]Member, error) {
	memberList, err := e.membersAPI.List(context.Background())
	if err != nil {
		return []Member{}, err
	}

	var members []Member
	for _, m := range memberList {
		members = append(members, Member{
			ID:         m.ID,
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
			ClientURLs: m.ClientURLs,
		})
	}

	return members, nil
}

func (e *EtcdClient) MemberAdd(peerURL string) (Member, error) {
	m, err := e.membersAPI.Add(context.Background(), peerURL)
	if err != nil {
		return Member{}, err
	}
	return Member{
		ID:         m.ID,
		Name:       m.Name,
		PeerURLs:   m.PeerURLs,
		ClientURLs: m.ClientURLs,
	}, nil
}
