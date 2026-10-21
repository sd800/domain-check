# Command-specific URL handling for Domain Check from Registries.
alias domain='noglob domain'

_domain_paste_context() {
    emulate -L zsh
    local left='^[[:space:]]*((command|noglob)[[:space:]]+)?domain([[:space:]]+(-{0,2}(en|zh|auto)))*[[:space:]]+$'
    local right='^([[:space:]]+(-{0,2}(en|zh|auto)))*[[:space:]]*$'
    [[ ${(L)LBUFFER} =~ $left && ${(L)RBUFFER} =~ $right ]]
}

_domain_prepare_paste() {
    emulate -L zsh
    REPLY=$1
    _domain_paste_context || return 1
    [[ $1 =~ '^[[:space:]]*[[:alpha:]][[:alnum:]+.-]*://' || $1 =~ '^[[:space:]]*[^[:space:]]+[.][^[:space:]]+' ]] || return 1
    # Quote as data, including substitutions, separators, glob characters,
    # spaces and newlines. Never evaluate the pasted content here.
    REPLY=${(q-)1}
}

_domain_url_paste() {
    emulate -L zsh
    if ! _domain_paste_context || [[ -n $NUMERIC || $REGION_ACTIVE != 0 || $LASTWIDGET == *vi-set-buffer ]]; then
        zle _domain_previous_bracketed_paste -- "$@"
        return
    fi
    local pasted REPLY
    zle .bracketed-paste pasted || return
    if _domain_prepare_paste "$pasted"; then
        pasted=$REPLY
    fi
    local start=$#LBUFFER
    LBUFFER+=$pasted
    YANK_START=$start
    YANK_END=$#LBUFFER
    zle -f yank
}

# Preserve other commands' paste widgets, and make repeated sourcing harmless.
if [[ -o interactive && ${widgets[bracketed-paste]} != user:_domain_url_paste ]]; then
    zle -A bracketed-paste _domain_previous_bracketed_paste
    zle -N bracketed-paste _domain_url_paste
fi
return 0
