package main

// imports //////////////////////////////////////

import (
  "os"
  _ "io"
  "fmt"
  "bufio"
  "regexp"
  "strings"
  "strconv"

  "github.com/spf13/cobra"
  "github.com/rs/zerolog"
  "github.com/coreos/go-systemd/journal"
  "github.com/satori/go.uuid"
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

      var messages []string

      if len(args) == 0 {
        stat, _ := os.Stdin.Stat()
        if (stat.Mode() & os.ModeCharDevice) == 0 {
          logger.Info().Msg("STDIN is open")

          in := bufio.NewScanner(os.Stdin)
          for in.Scan() {
            message := strings.TrimSpace(in.Text())
            messages = append(messages, message)
            logger.Info().
              Str("line", message).
              Msg("Read from STDIN")
          }
          if in.Err() != nil {
            logger.Info().
              Str("error", fmt.Sprint(in.Err())).
              Msg("Encountered an error while reading from STDIN")
          }


        } else {
          cmd.Help()
          os.Exit(ExitStatusArgument)
        }

      } else {
        // determine message from command line arguments, which is always
        // the first argument
        messages = append(messages, strings.TrimSpace(args[0]))
      }

      for _, message := range messages {
        logger.Info().
          Str("message", message).
          Msg("Determined message")

        // create logger for calling process, based on priority
        kv := map[string]string{}
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

            if fstderr {
              // if writing to stderr, we need to infer
              // type and build zerlogger event chain

              // attempt to parse float, bool, int and then fallback
              // to string;
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
        }

        if _, ok := kv["MESSAGE_ID"]; !ok {
          // if not already passed, create a message id, which will
          // be used primarily by journald but is also packaged into
          // the stderr payload
          kv["MESSAGE_ID"] = fmt.Sprint(uuid.NewV4())
          logger.Info().
            Str("message_id", kv["MESSAGE_ID"]).
            Msg("Determined message id")

          event = event.Str("MESSAGE_ID", kv["MESSAGE_ID"])
        }

        // call Msg to trigger event
        if fstderr {
          event.Msg(message)
        }

        // check if journald is available and if the case,
        // write to it
        if journal.Enabled() {
          logger.Info().Msg("Journald is available")
          err := journal.Send(message, journal.Priority(level), kv)
          if err != nil {
            logger.Info().
              Str("error", err.Error()).
              Msg("Failed to send message to journald")
          }
        }
      }
    },
  }

  root.PersistentFlags().BoolVarP(
    &fpid, "pid", "i", false, "Log the process id of the logger process with each line.",
  )
  root.PersistentFlags().BoolVarP(
    &fstderr, "stderr", "s", false, "Log the message to standard error, as well as the system log.",
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
