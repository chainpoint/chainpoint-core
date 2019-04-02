package abci

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

func TestValidateRepChainItemHash(t *testing.T) {
	assert := assert.New(t)
	app := DeclareABCI()
	repChainItem := types.RepChainItem{
		ID:              34559,
		CalBlockHeight:  765756,
		CalBlockHash:    "9cab80484288b0467044600111c823d8a67bd3bba9063c9b7bb6ddd6f506baf2",
		PrevRepItemHash: "d6f253786233e9ee0e91f894cc51d7c79a5455dbc7ee509c5b9ca7bc669e02c1",
		HashIDNode:      "3de6d66e-4bd8-11e9-8646-d663bd873d93",
		RepItemHash:     "c9ee7f0b005eb6ef26dc09eb1c99f0402ef2fdb3acd214634e8b70a21bcab465",
	}
	hash, err := app.ValidateRepChainItemHash(repChainItem)
	util.LogError(err)
	assert.Equal(hash, "c9ee7f0b005eb6ef26dc09eb1c99f0402ef2fdb3acd214634e8b70a21bcab465", "hashes should be equal")
}
