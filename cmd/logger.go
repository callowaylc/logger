package main

// imports //////////////////////////////////////

import (
  "os"
  "fmt"
  "bufio"
  "strings"
  "strconv"

  "github.com/spf13/cobra"
  "github.com/rs/zerolog"
  "github.com/coreos/go-systemd/journal"
  "github.com/satori/go.uuid"
  _ "github.com/rs/zerolog/log"


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

        // prepend/push a "nil" argument onto args; attribute parsing
        // occurs args[1+N] where N >=0; ie, we need to fulfill the
        // expectations set fourth that attributes will occur after
        // the initial argument
        args = append([]string{ "" }, args...)

      } else if len(args) > 0 {
        // determine message from command line arguments, which is always
        // the first argument
        messages = append(messages, strings.TrimSpace(args[0]))

      } else {
        // if no stdin and no arguments then we display our help message
        // and exit with a failed status code
        cmd.Help()
        os.Exit(ExitStatusArgument)
      }

      for _, message := range messages {
        logger.Info().
          Str("message", message).
          Msg("Determined message")

        // create logger for calling process, based on priority
        kv := map[string]string{}
        var level zerolog.Level = zerolog.InfoLevel

        // attempt to use zerologger to parse priorty; if that
        // fails, we exit with a return code of 1
        level, err := log.ParseLevel(fpriority)
        if err != nil {
          logger.Info().
            Str("priority", fpriority).
            Str("error", err.Error()).
            Msg("Failed to determine level")

          // mirrors behavior of linux logger
          fmt.Fprintf(
            os.Stderr,
            "logger: unknown priority name: %s\n",
            fpriority,
          )

          os.Exit(1)
        }
        logger.Info().
          Int("level", int(level)).
          Str("priority", fpriority).
          Msg("Determined event level")

        // create logger and event, with minimum accepted leve,
        // if environment variable "PRIORITY" exists
        minimum := zerolog.InfoLevel
        v, ok := os.LookupEnv("PRIORITY"); if ok {
          minimum, err = log.ParseLevel(v)
          if err != nil {
            logger.Info().
              Str("priority", fpriority).
              Str("error", err.Error()).
              Msg("Failed to determine priority")

            // mirrors behavior of linux logger
            fmt.Fprintf(
              os.Stderr,
              "logger: unknown priority name: %s\n",
              v,
            )

            os.Exit(1)
          }
        }
        logger.Info().
          Int("level", int(level)).
          Int("priority", int(minimum)).
          Msg("Determined priority level")

        l := zerolog.New(os.Stderr).Level(minimum)
        var event *zerolog.Event = l.WithLevel(level)
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

        // check if environment variable _GROUP_ID exists,
        // in which case we can flag existence
        v, ok = os.LookupEnv("_GROUP_ID"); if ok {
          logger.Info().
            Str("method", "environment").
            Str("group_id", v).
            Msg("Group ID was passed")

          kv["GROUP_ID"] = v
          event = event.Str("GROUP_ID", kv["GROUP_ID"])
        }

        // check if message_id has been passed as an argument
        // if the case, we will use this value as a message
        // primary key
        exists := false
        for k, v := range kv {
          if strings.ToUpper(k) == "MESSAGE_ID" {
            logger.Info().
              Str("method", "argument").
              Str("message_id", v).
              Msg("Message ID was passed")

            delete(kv, k)
            kv[strings.ToUpper(k)] = v
            exists = true
            break
          }
        }
        if !exists {
          // if not already passed, create a message id, which will
          // be used primarily by journald but is also packaged into
          // the stderr payload
          kv["MESSAGE_ID"] = fmt.Sprint(uuid.NewV4())
        }
        event = event.Str("MESSAGE_ID", kv["MESSAGE_ID"])
        logger.Info().
          Str("message_id", kv["MESSAGE_ID"]).
          Msg("Determined message id")

        // call Msg to trigger event
        if fstderr {
          event.Msg(message)
        }

        // check if journald is available and if the case,
        // write to it
        if journal.Enabled() {
          logger.Info().Msg("Journald is available")

          // journald priorities are represented by different
          // priority integer values than zerolog; we need
          // to define a map, in order to perform a converstion,
          // when sending to journald
          pmap := map[zerolog.Level]journal.Priority{
            zerolog.DebugLevel: journal.PriDebug,
            zerolog.InfoLevel: journal.PriInfo,
            zerolog.WarnLevel: journal.PriWarning,
            zerolog.ErrorLevel: journal.PriErr,
            zerolog.FatalLevel: journal.PriCrit,
            zerolog.PanicLevel: journal.PriEmerg,
          }

          // journald fields must be uppercased
          fkv := map[string]string{}
          for key, value := range kv {
            fkv[strings.ToUpper(key)] = value
          }
          logger.Info().
            Str("formatted_key_value", fmt.Sprint(fkv)).
            Msg("Formatted keys for journald")

          err := journal.Send(message, journal.Priority(pmap[level]), fkv)
          if err != nil {
            logger.Error().
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

func trace(function string) string {
  return fmt.Sprintf(
    "%s#%s.%s@%s", pkg.PROJECT, "main", function, "logger",
  )
}
