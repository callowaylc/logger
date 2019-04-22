package pkg

import (
	"fmt"
)

// constants ////////////////////////////////////

const PROJECT string = "github.com/callowaylc/logger"

// functions ////////////////////////////////////


func Trace(function, pkg string) string {
  return fmt.Sprintf(
    "%s#%s.%s@%s", PROJECT, "main", function, pkg,
  )
}
