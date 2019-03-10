package main

// imports //////////////////////////////////////

import (
  "os"
  "fmt"
  "regexp"
  "strings"
  "strconv"

  "github.com/spf13/cobra"
  "github.com/rs/zerolog"
  "github.com/coreos/go-systemd/journal"
  zlog "github.com/rs/zerolog/log"


  "github.com/callowaylc/logger/pkg"
  "github.com/callowaylc/logger/pkg/log"
)

// constants ////////////////////////////////////

const ExitStatusArgument int = 3
const ExitStatusFormat int = 4

// main /////////////////////////////////////////

func init() {
  log.Init()
}

func main() {
  logger := log.Logger(trace("main"))
  logger.Info().Msg("Enter")
  defer logger.Info().Msg("Exit")

  var fpid bool
  var fstderr bool
  var fjson bool
  var fpriority string
  var ftag string
  var ffile string

  root := &cobra.Command{
    Use: "logger [-is] [-f file] [-p pri] [-t tag] [message ...]",

    // define logger behavior, which is to parse message and
    // write to stderr
    Run: func(cmd *cobra.Command, args []string) {
      logger := log.Logger(trace("main.Run"))
      logger.Info().
        Str("args", fmt.Sprint(args)).
        Msg("Enter")
      defer logger.Info().Msg("Exit")

      if len(args) == 0 {
        cmd.Help()
        os.Exit(ExitStatusArgument)
      }

      // determine message, which is always
      // the first argument
      message := args[0]
      logger.Info().
        Str("mmessage", message).
        Msg("Determined message")

      // create logger for calling process, based on priority
      var level zerolog.Level = zerolog.InfoLevel
      switch {
      case match(`(?i)debug`, fpriority):
        level = zerolog.DebugLevel
      case match(`(?i)notice`, fpriority):
        level = zerolog.InfoLevel
      case match(`(?i)warn`, fpriority):
        level = zerolog.WarnLevel
      case match(`(?i)err`, fpriority):
        level = zerolog.ErrorLevel
      case match(`(?i)(crit|alert)`, fpriority):
        level = zerolog.FatalLevel
      case match(`(?i)emerg`, fpriority):
        level = zerolog.PanicLevel
      }
      logger.Info().
        Int("level", int(level)).
        Str("priority", fpriority).
        Msg("Determined priority level")

      // create event with determined level
      var event *zerolog.Event = zlog.WithLevel(level)
      logger.Info().Msg("Created log event")

      // parse arguments that will makeup the structured log line;
      // these should be all arguments that fall after the first, ie
      // the message
      kv := map[string]string{}

      if len(args) > 1 {
        for _, pair := range args[1:] {
          logger.Info().
            Str("raw", pair).
            Msg("Evaluating argument pair")

          // pairs must be passed as key=value or
          // we panic out
          result := strings.SplitN(pair, "=", 2)
          if len(result) != 2 {
            logger.Error().
              Str("raw", pair).
              Str("result", fmt.Sprint(result)).
              Msg("Failed to pass pair as key=value")
            os.Exit(ExitStatusFormat)
          }
          k, v := result[0], result[1]
          kv[k] = v
          logger.Info().
            Str("raw", pair).
            Str("result", fmt.Sprint(result)).
            Str("key", k).
            Str("value", v).
            Msg("Determined key/value pair")

          // attempt to parse float, bool, int and then fallback
          // to string
          var value interface{}
          var err error

          value, err = strconv.ParseInt(v, 10, 64)
          if err != nil || strings.Contains(v, ".") {
            value, err = strconv.ParseFloat(v, 64)
          }
          if err != nil {
            value, err = strconv.ParseBool(v)
          }
          if err != nil {
            value = fmt.Sprint(v)
          }
          logger.Info().
            Str("raw", pair).
            Str("value", fmt.Sprint(value)).
            Str("type", fmt.Sprintf("%T", value)).
            Msg("Determined value type")

          // now use type assertion against a switch statement
          // to chain an event based on type
          switch v := value.(type) {
          case int64:
            event = event.Int64(k, v)
          case float64:
            event = event.Float64(k, v)
          case bool:
            event = event.Bool(k, v)
          default:
            event = event.Str(k, fmt.Sprint(v))
          }
          logger.Info().
            Str("raw", pair).
            Msg("Chained event")
        }
      }

      // call Msg to trigger event
      event.Msg(message)

      // check if journald is available and if the case,
      // write to it
      if journal.Enabled() {
        logger.Info().
          Msg("Journald is available")
        journal.Send(message, journal.Priority(level), kv)
      }
    },
  }

  root.PersistentFlags().BoolVarP(
    &fpid, "pid", "i", false, "Log the process id of the logger process with each line.",
  )
  root.PersistentFlags().BoolVarP(
    &fstderr, "stderr", "s", false, "NOP",
  )
  root.PersistentFlags().BoolVarP(
    &fjson, "json", "j", true, "Express log message as json.",
  )
  root.PersistentFlags().StringVarP(
    &ffile, "file", "f", "", "NOP",
  )
  root.PersistentFlags().StringVarP(
    &fpriority, "priority", "p", "user.notice", "Enter the message with the specified priority.",
  )
  root.PersistentFlags().StringVarP(
    &ftag, "tag", "t", "", "Mark every line in the log with the specified tag.",
  )
  root.Execute()
}

func match(pattern, subject string) bool {
  if ok, _ := regexp.MatchString(pattern, subject); ok {
    return true
  }

  return false
}

func trace(function string) string {
  return fmt.Sprintf(
    "%s#%s.%s@%s", pkg.PROJECT, "main", function, "logger",
  )
}
