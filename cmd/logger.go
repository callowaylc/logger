package main

// imports //////////////////////////////////////
import (
  "os"
  "fmt"
  "regexp"

  "github.com/spf13/cobra"
  "github.com/rs/zerolog"
  //zlog "github.com/rs/zerolog/log"


  "github.com/callowaylc/logger/pkg"
  "github.com/callowaylc/logger/pkg/log"
)

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
        os.Exit(1)
      }

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

      // parse arguments that will makeup the structured log line

      //zlog.Logger = zlog.Output(zlog.ConsoleWriter{Out: os.Stderr})


      //zlog.Info().Str("args", fmt.Sprint(args)).Msg("Submitted arguments")
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
