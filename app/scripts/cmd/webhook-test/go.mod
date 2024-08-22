module webhook-test

go 1.21.3

replace github.com/plandex/plandex/shared => ../../../shared

replace plandex-server => ../../../server

require (
	github.com/plandex/plandex/shared v0.0.0-00010101000000-000000000000
	plandex-server v0.0.0-00010101000000-000000000000
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.17.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sashabaranov/go-openai v1.24.0 // indirect
	github.com/smacker/go-tree-sitter v0.0.0-20240423010953-8ba036550382 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/image v0.17.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
)
