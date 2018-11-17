package pkg

// imports //////////////////////////////////////

import(
  "github.com/rs/zerolog"
  "github.com/rs/zerolog/log"
)

// functions ////////////////////////////////////

func InitLogs() { }

func Logger(trace string, level zerlog.Level) *zerolog.Event {
  return log.
    WithLevel(level).
    With().
    Str("trace", trace).
    Logger()
}
