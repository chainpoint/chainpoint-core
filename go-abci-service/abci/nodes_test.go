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
		RepItemHash:     "aee3c63d9f9bbf389181f4e81b36d26bb4d2497dd4e763bf05442a3f2bfb1370",
	}
	hash, err := app.ValidateRepChainItemHash(repChainItem)
	util.LogError(err)
	assert.Equal(hash, "aee3c63d9f9bbf389181f4e81b36d26bb4d2497dd4e763bf05442a3f2bfb1370", "hashes should be equal")
}
