package fault

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func alphabetClaim(index uint64, letter string) common.Hash {
	return crypto.Keccak256Hash(BuildAlphabetPreimage(index, letter))
}

// TestSolver_NextMove_Opponent tests the [Solver] NextMove function
// with an [fault.AlphabetProvider] as the [TraceProvider].
func TestSolver_NextMove_Opponent(t *testing.T) {
	// Construct the solver.
	maxDepth := 3
	canonicalProvider := NewAlphabetProvider("abcdefgh", uint64(maxDepth))
	solver := NewSolver(maxDepth, canonicalProvider)

	// The following claims are created using the state: "abcdexyz".
	// The responses are the responses we expect from the solver.
	indices := []struct {
		claim    Claim
		response ClaimData
	}{
		{
			Claim{
				ClaimData: ClaimData{
					Value:    alphabetClaim(7, "z"),
					Position: NewPosition(0, 0),
				},
				// Root claim has no parent
			},
			ClaimData{
				Value:    alphabetClaim(3, "d"),
				Position: NewPosition(1, 0),
			},
		},
		{
			Claim{
				ClaimData: ClaimData{
					Value:    alphabetClaim(3, "d"),
					Position: NewPosition(1, 0),
				},
				Parent: ClaimData{
					Value:    alphabetClaim(7, "h"),
					Position: NewPosition(0, 0),
				},
			},
			ClaimData{
				Value:    alphabetClaim(5, "f"),
				Position: NewPosition(2, 2),
			},
		},
		{
			Claim{
				ClaimData: ClaimData{
					Value:    alphabetClaim(5, "x"),
					Position: NewPosition(2, 2),
				},
				Parent: ClaimData{
					Value:    alphabetClaim(7, "h"),
					Position: NewPosition(1, 1),
				},
			},
			ClaimData{
				Value:    alphabetClaim(4, "e"),
				Position: NewPosition(3, 4),
			},
		},
	}

	for _, test := range indices {
		res, err := solver.NextMove(test.claim, false)
		require.NoError(t, err)
		require.Equal(t, test.response, res.ClaimData)
	}
}

func TestNoMoveAgainstOwnLevel(t *testing.T) {
	maxDepth := 3
	mallory := NewAlphabetProvider("abcdepqr", uint64(maxDepth))
	solver := NewSolver(maxDepth, mallory)

	claim := Claim{
		ClaimData: ClaimData{
			Value:    alphabetClaim(7, "z"),
			Position: NewPosition(0, 0),
		},
		// Root claim has no parent
	}

	move, err := solver.NextMove(claim, true)
	require.Nil(t, move)
	require.Nil(t, err)
}

func TestAttemptStep(t *testing.T) {
	maxDepth := 3
	canonicalProvider := &alphabetWithProofProvider{NewAlphabetProvider("abcdefgh", uint64(maxDepth))}
	solver := NewSolver(maxDepth, canonicalProvider)
	_, _, middle, bottom := createTestClaims()

	zero := Claim{
		ClaimData: ClaimData{
			// Zero value is a purposely disagree with claim value "a"
			Position: NewPosition(3, 0),
		},
	}

	step, err := solver.AttemptStep(bottom, false)
	require.NoError(t, err)
	require.Equal(t, bottom, step.LeafClaim)
	require.True(t, step.IsAttack)
	require.Equal(t, step.PreState, BuildAlphabetPreimage(3, "d"))
	require.Equal(t, step.ProofData, []byte{3})

	_, err = solver.AttemptStep(middle, false)
	require.Error(t, err)

	step, err = solver.AttemptStep(zero, false)
	require.NoError(t, err)
	require.Equal(t, zero, step.LeafClaim)
	require.True(t, step.IsAttack)
	require.Equal(t, canonicalProvider.AbsolutePreState(), step.PreState)
}

func TestAttempStep_AgreeWithClaimLevel_Fails(t *testing.T) {
	maxDepth := 3
	canonicalProvider := NewAlphabetProvider("abcdefgh", uint64(maxDepth))
	solver := NewSolver(maxDepth, canonicalProvider)
	_, _, middle, _ := createTestClaims()

	step, err := solver.AttemptStep(middle, true)
	require.Error(t, err)
	require.Equal(t, StepData{}, step)
}

type alphabetWithProofProvider struct {
	*AlphabetProvider
}

func (a *alphabetWithProofProvider) GetPreimage(i uint64) ([]byte, []byte, error) {
	preimage, _, err := a.AlphabetProvider.GetPreimage(i)
	if err != nil {
		return nil, nil, err
	}
	return preimage, []byte{byte(i)}, nil
}
