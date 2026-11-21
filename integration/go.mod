module github.com/tophergopher/genericmongo/integration

go 1.26.0

toolchain go1.27.4

require (
	github.com/stretchr/testify v1.12.1
	github.com/tophergopher/dbtest/mongotest v0.3.0
	github.com/tophergopher/genericmongo v0.0.0
	go.mongodb.org/mongo-driver v1.17.10
)

// The library comes from the working tree, so a change to it is tested here
// before it is tagged.
replace github.com/tophergopher/genericmongo => ..
