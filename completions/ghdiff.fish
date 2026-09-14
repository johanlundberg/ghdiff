# fish completion for ghdiff

function __ghdiff_needs_command
    set cmd (commandline -opc)
    if test (count $cmd) -eq 1
        return 0
    end
    return 1
end

function __ghdiff_using_flag
    set cmd (commandline -opc)
    set flag $argv[1]
    contains -- $flag $cmd
end

# Global options
complete -c ghdiff -l port -x -a '0 8080 9090 3000 4000' -d 'HTTP server port (0 = auto)'
complete -c ghdiff -l host -x -a 'localhost 127.0.0.1 0.0.0.0' -d 'HTTP server host'
complete -c ghdiff -l no-open -d 'Do not open browser automatically'
complete -c ghdiff -l mode -x -a 'split unified' -d 'Initial view mode'
complete -c ghdiff -l version -d 'Print version and exit'
complete -c ghdiff -l help -d 'Print usage information'

# Positional arguments: git refs (when no flag is being completed)
complete -c ghdiff -n '__ghdiff_needs_command' -x -a '(__fish_git_branches)' -d 'Git branch'
complete -c ghdiff -n '__ghdiff_needs_command' -x -a '(__fish_git_tags)' -d 'Git tag'
complete -c ghdiff -n '__ghdiff_needs_command' -a '.' -d 'Working directory mode'
complete -c ghdiff -n '__ghdiff_needs_command' -a '-' -d 'Read diff from stdin'