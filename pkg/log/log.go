package log

// imports //////////////////////////////////////

import(
  "os"
  "regexp"

  "github.com/rs/zerolog"
)

// constants ////////////////////////////////////

// functions ////////////////////////////////////

func Init() { }

func Logger(trace string) zerolog.Logger {
  // determine reexp
  r := regexp.MustCompile(`^(.+?)#(.+?)\.(.+?)@(.+)$`)
  matches := r.FindStringSubmatch(trace)
  public, _ := regexp.MatchString("^[A-Z]", matches[3])

  // return json logger unless the env LOGLOGGER=true exists
  // at the time of call
  v, ok := os.LookupEnv("LOGLOGGER"); if ok && v == "true" {
    return zerolog.
      New(os.Stderr).
      With().
      Caller().
      Str("Trace", trace).
      Str("Project", matches[1]).
      Str("Package", matches[2]).
      Str("Function", matches[3]).
      Str("File", matches[4]).
      Bool("Public", public).
      Logger()
  }

  // otherwise return a nop logger
  return zerolog.Nop()
}
