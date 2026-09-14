# bash completion for ghdiff                                -*- shell-script -*-

_ghdiff()
{
    local cur prev words cword
    _init_completion || return

    # List of long options
    local opts="--port --host --no-open --mode --version --help"

    case $prev in
        --port)
            COMPREPLY=($(compgen -W "0 8080 9090 3000 4000 5000" -- "$cur"))
            return
            ;;
        --host)
            COMPREPLY=($(compgen -W "localhost 127.0.0.1 0.0.0.0" -- "$cur"))
            return
            ;;
        --mode)
            COMPREPLY=($(compgen -W "split unified" -- "$cur"))
            return
            ;;
    esac

    if [[ $cur == -* ]]; then
        COMPREPLY=($(compgen -W "$opts" -- "$cur"))
    else
        # Complete with git refs (branches, tags, etc.)
        if command -v git &>/dev/null; then
            local git_refs
            git_refs=$(git rev-parse --git-dir 2>/dev/null) && \
                COMPREPLY=($(compgen -W "$(git for-each-ref --format='%(refname:short)' refs/heads/ refs/tags/ 2>/dev/null) HEAD" -- "$cur"))
        fi
    fi
} &&
complete -F _ghdiff ghdiff

# ex: ts=4 sw=4 et filetype=sh