package common

var (

	// ProposalDraftDataBucketName is the name of the DB bucket used to house the
	// proposal releated draft data and draft hash.
	ProposalDraftDataBucketName = []byte("proposaldraftdata")

	// Tx3IndexBucketName is the key of the tx3 index and the DB bucket used
	// to house it.
	Tx3IndexBucketName = []byte("tx3hash")

	// Tx3IndexValue is placeholder for tx3 index
	Tx3IndexValue = []byte{1}
)
