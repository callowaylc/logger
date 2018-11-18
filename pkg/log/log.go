package log

// imports //////////////////////////////////////

import(
  "regexp"

  "github.com/rs/zerolog"
  "github.com/rs/zerolog/log"
)

// types ////////////////////////////////////////

type Level zerolog.Level

// constants ////////////////////////////////////

const DebugLevel zerolog.Level = zerolog.DebugLevel
const InfoLevel zerolog.Level = zerolog.InfoLevel

// functions ////////////////////////////////////

func Init() { }

func Logger(trace string, level zerolog.Level) *zerolog.Event {
  r := regexp.MustCompile(`^(.+?)#(.+?)\.(.+?)@(.+)$`)
  matches := r.FindStringSubmatch(trace)

  logger := log.
    With().
    Str("Trace", trace).
    Str("Project", matches[1]).
    Str("Package", matches[2]).
    Str("Function", matches[3]).
    Str("File", matches[4]).
    Logger()

  return logger.WithLevel(level)
}
