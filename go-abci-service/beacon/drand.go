package beacon

import (
	"context"
	"errors"
	"fmt"
	"github.com/drand/drand/key"
	"os"

	"github.com/drand/drand/client"
	"github.com/drand/drand/client/grpc"
)

func getPublicRandomness() (error, client.Result) {
	certPath := ""
	ids, err := getNodes()
	if err != nil {
		return err, nil
	}
	group, err := getGroup()
	if err != nil {
		return err, nil
	}
	if group.PublicKey == nil {
		return errors.New("drand: group file must contain the distributed public key"), nil
	}

	var resp client.Result
	var foundCorrect bool
	for _, id := range ids {
		cli, err := grpc.New(id.Addr, certPath, !id.TLS)
		if err != nil {
			fmt.Fprintf(os.Stderr, "drand: could not connect to %s: %s", id.Addr, err)
			break
		}

		resp, err = cli.Get(context.Background(), 0)

		if err == nil {
			foundCorrect = true
			break
		}
		fmt.Fprintf(os.Stderr, "drand: could not get public randomness from %s: %s", id.Addr, err)
	}
	if !foundCorrect {
		return errors.New("drand: could not verify randomness"), nil
	}
	return nil, resp
}

func getNodes() ([]*key.Node, error) {
	group, err := getGroup()
	if err != nil {
		return nil, err
	}
	var ids []*key.Node
	gids := group.Nodes
	ids = gids
	if len(ids) == 0 {
		return nil, errors.New("no nodes specified with --nodes are in the group file")
	}
	return ids, nil
}

//TODO: load group path dynamically
func getGroup() (*key.Group, error) {
	g := &key.Group{}
	groupPath := "/go/src/github.com/chainpoint/chainpoint-core/go-abci-service/group.toml"
	if err := key.Load(groupPath, g); err != nil {
		return nil, fmt.Errorf("drand: error loading group file: %s", err)
	}
	return g, nil
}
