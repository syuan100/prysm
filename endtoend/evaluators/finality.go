package evaluators

import (
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"google.golang.org/grpc"
)

// FinalizationOccurs is an evaluator to make sure finalization is performing as it should.
// Requires to be run after at least 4 epochs have passed.
var FinalizationOccurs = types.Evaluator{
	Name:       "finalizes_at_epoch_%d",
	Policy:     afterNthEpoch(3),
	Evaluation: finalizationOccurs,
}

func finalizationOccurs(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := eth.NewBeaconChainClient(conn)
	chainHead, err := client.GetChainHead(context.Background(), &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}
	currentEpoch := chainHead.HeadEpoch
	finalizedEpoch := chainHead.FinalizedEpoch

	expectedFinalizedEpoch := currentEpoch - 2
	if expectedFinalizedEpoch != finalizedEpoch {
		return fmt.Errorf(
			"expected finalized epoch to be %d, received: %d",
			expectedFinalizedEpoch,
			finalizedEpoch,
		)
	}
	previousJustifiedEpoch := chainHead.PreviousJustifiedEpoch
	currentJustifiedEpoch := chainHead.JustifiedEpoch
	if previousJustifiedEpoch+1 != currentJustifiedEpoch {
		return fmt.Errorf(
			"there should be no gaps between current and previous justified epochs, received current %d and previous %d",
			currentJustifiedEpoch,
			previousJustifiedEpoch,
		)
	}
	if currentJustifiedEpoch+1 != currentEpoch {
		return fmt.Errorf(
			"there should be no gaps between current epoch and current justified epoch, received current %d and justified %d",
			currentEpoch,
			currentJustifiedEpoch,
		)
	}
	return nil
}
