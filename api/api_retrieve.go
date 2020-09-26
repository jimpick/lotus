package api

import (
	"context"
)

type Retrieve interface {
	// ClientRetrieve initiates the retrieval of a file, as specified in the order.
	ClientRetrieve(ctx context.Context, order RetrievalOrder, ref *FileRef) error
}
