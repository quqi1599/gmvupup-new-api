package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstreamClientRejectionProvenance(t *testing.T) {
	for _, options := range [][]NewAPIErrorOptions{nil, {ErrOptionWithUpstreamResponse(), ErrOptionWithSkipRetry()}} {
		err := NewErrorWithStatusCode(errors.New("fixture"), ErrorCodeBadResponseStatusCode, 400, options...)
		err.MarkUpstreamClientRejected()
		require.False(t, IsUpstreamClientRejected(err))
	}
	err := WithOpenAIError(OpenAIError{Message: "fixture", Code: string(ErrorCodeUpstreamClientRejected)}, 400, ErrOptionWithUpstreamResponse())
	require.False(t, IsUpstreamClientRejected(err), "untrusted upstream code cannot mark the local classification")
	err.MarkUpstreamClientRejected()
	require.True(t, IsUpstreamClientRejected(err))
	wrapped := NewError(err, ErrorCodeBadResponse, ErrOptionWithSkipRetry())
	require.True(t, IsUpstreamClientRejected(wrapped))
	require.True(t, IsSkipRetryError(wrapped), "later replay barriers remain authoritative")
}
