package arg_cmd

import (
	"fmt"

	appctx "github.com/nixys/nxs-go-appctx/v2"
)

func TestConfig(appCtx *appctx.AppContext) error {
	fmt.Println("The configuration is correct.")
	return nil
}
