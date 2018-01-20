package sharding

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "geth" // Used to determine the ipc name.
)

// Client for sharding. Communicates to geth node via JSON RPC.
type Client struct {
	endpoint  string             // Endpoint to JSON RPC
	client    *ethclient.Client  // Ethereum RPC client.
	keystore  *keystore.KeyStore // Keystore containing the single signer
	ctx       *cli.Context       // Command line context
	networkID uint64             // Ethereum network ID
}

// MakeShardingClient for interfacing with geth full node.
func MakeShardingClient(ctx *cli.Context) *Client {
	path := node.DefaultDataDir()
	if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		path = ctx.GlobalString(utils.DataDirFlag.Name)
	}
	endpoint := fmt.Sprintf("%s/%s.ipc", path, clientIdentifier)

	config := &node.Config{
		DataDir: path,
	}
	scryptN, scryptP, keydir, err := config.AccountConfig()
	if err != nil {
		panic(err) // TODO(prestonvanloon): handle this
	}
	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	networkID := uint64(1)
	if ctx.GlobalIsSet(utils.NetworkIdFlag.Name) {
		networkID = ctx.GlobalUint64(utils.NetworkIdFlag.Name)
	}

	return &Client{
		endpoint:  endpoint,
		keystore:  ks,
		ctx:       ctx,
		networkID: networkID,
	}
}

// Start the sharding client.
// * Connects to node.
// * Verifies or deploys the validator management contract.
func (c *Client) Start() error {
	log.Info("Starting sharding client")
	rpcClient, err := dialRPC(c.endpoint)
	if err != nil {
		return err
	}
	c.client = ethclient.NewClient(rpcClient)
	defer rpcClient.Close()
	if err := c.verifyVMC(); err != nil {
		return err
	}

	// TODO: Wait to be selected as collator in goroutine?

	return nil
}

// Wait until sharding client is shutdown.
func (c *Client) Wait() {
	// TODO: Blocking lock.
}

// dialRPC endpoint to node.
func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	}
	return rpc.Dial(endpoint)
}

// UnlockAccount will unlock the specified account using utils.PasswordFileFlag or empty string if unset.
func (c *Client) unlockAccount(account accounts.Account) error {
	pass := ""

	if c.ctx.GlobalIsSet(utils.PasswordFileFlag.Name) {
		blob, err := ioutil.ReadFile(c.ctx.GlobalString(utils.PasswordFileFlag.Name))
		if err != nil {
			return fmt.Errorf("unable to read account password contents in file %s. %v", utils.PasswordFileFlag.Value, err)
		}
		// TODO: Use bufio.Scanner or other reader that doesn't include a trailing newline character.
		pass = strings.Trim(string(blob), "\n") // Some text files end in new line, remove with strings.Trim.
	}

	return c.keystore.Unlock(account, pass)
}
