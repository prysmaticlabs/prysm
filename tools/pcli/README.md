## Pcli (Prysm CLI)

This is a utility to help users perform eth2 specific actions.

### Usage

*Name:*  
   **pcli** - A command line utility to run eth2 specific actions

*Usage:*  
   pcli [global options] command [command options] [arguments...]

*Commands:*
     help, h  Shows a list of commands or help for one command
   state-transition:
     state-transition  Subcommand to run manual state transitions


*Flags:*  
   --help, -h     show help (default: false)
   --version, -v  print the version (default: false)

*State Transition Subcommand:*
   pcli state-transition - Subcommand to run manual state transitions

*State Transition Usage:*:
   pcli state-transition [command options] [arguments...]


*State Transition Flags:*
   --blockPath value              Path to block file(ssz)
   --preStatePath value           Path to pre state file(ssz)
   --expectedPostStatePath value  Path to expected post state file(ssz)
   --help, -h                     show help (default: false)



### Example

To use pcli manual state transition:

```
bazel run //tools/pcli:pcli -- state-transition --blockPath /path/to/block --preStatePath /path/to/state
```

