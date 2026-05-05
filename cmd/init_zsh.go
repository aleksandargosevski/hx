package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Print shell integration snippets",
}

var initZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Print zsh integration snippet",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(zshSnippet)
	},
}

func init() {
	initCmd.AddCommand(initZshCmd)
}

const zshSnippet = `# hx - History Extended
# Add to .zshrc: eval "$(hx init zsh)"

HISTFILE="${HISTFILE:-$HOME/.zsh_history}"
HISTSIZE=100000
SAVEHIST=100000

setopt EXTENDED_HISTORY
setopt HIST_EXPIRE_DUPS_FIRST
setopt HIST_IGNORE_ALL_DUPS
setopt HIST_IGNORE_SPACE
setopt HIST_SAVE_NO_DUPS
setopt SHARE_HISTORY

# Record commands to hx database
__hx_preexec() {
  __hx_cmd="$1"
  __hx_start=$EPOCHSECONDS
  __hx_dir="$PWD"
}

# Detect commands containing secrets/tokens — skip recording
__hx_is_sensitive() {
  local cmd="$1"
  # AWS
  [[ "$cmd" == *AKIA[0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z]* ]] && return 0
  [[ "$cmd" == *AWS_SECRET_ACCESS_KEY=* ]] && return 0
  [[ "$cmd" == *AWS_SESSION_TOKEN=* ]] && return 0
  # GitHub tokens
  [[ "$cmd" == *ghp_* ]] && return 0
  [[ "$cmd" == *gho_* ]] && return 0
  [[ "$cmd" == *ghu_* ]] && return 0
  [[ "$cmd" == *ghs_* ]] && return 0
  [[ "$cmd" == *github_pat_* ]] && return 0
  # Slack
  [[ "$cmd" == *xoxb-* ]] && return 0
  [[ "$cmd" == *xoxp-* ]] && return 0
  [[ "$cmd" == *xoxa-* ]] && return 0
  # Stripe
  [[ "$cmd" == *sk_live_* ]] && return 0
  [[ "$cmd" == *rk_live_* ]] && return 0
  # Generic secret patterns
  [[ "$cmd" == *--password=* ]] && return 0
  [[ "$cmd" == *--password\ * ]] && return 0
  [[ "$cmd" == *--token=* ]] && return 0
  [[ "$cmd" == *--token\ * ]] && return 0
  [[ "$cmd" == *API_KEY=* ]] && return 0
  [[ "$cmd" == *SECRET_KEY=* ]] && return 0
  [[ "$cmd" == *PRIVATE_KEY=* ]] && return 0
  # NPM tokens
  [[ "$cmd" == *npm_* ]] && return 0
  # Netlify
  [[ "$cmd" == *nfp_* ]] && return 0
  return 1
}

__hx_precmd() {
  local exit_code=$?
  if [[ -n "$__hx_cmd" ]]; then
    # Skip sensitive commands containing secrets/tokens
    if __hx_is_sensitive "$__hx_cmd"; then
      __hx_cmd=""
      return
    fi
    local duration=0
    if [[ -n "$__hx_start" ]]; then
      duration=$(( EPOCHSECONDS - __hx_start ))
    fi
    command hx record \
      --command "$__hx_cmd" \
      --dir "$__hx_dir" \
      --exit-code "$exit_code" \
      --duration "$duration" &!
    __hx_cmd=""
  fi
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec __hx_preexec
add-zsh-hook precmd __hx_precmd

# --- Snippet state ---
typeset -a __hx_snippet_starts __hx_snippet_ends __hx_snippet_labels
typeset -i __hx_snippet_index=0
typeset -i __hx_snippet_active=0

# --- Ctrl+R widget — launch hx search ---
__hx_search() {
  local initial_query="$BUFFER"
  BUFFER=""
  local selected
  selected=$(command hx search --query "$initial_query" --cwd "$PWD")
  local ret=$?

  if [[ -z "$selected" ]]; then
    BUFFER="$initial_query"
    CURSOR=${#BUFFER}
    zle reset-prompt
    return $ret
  fi

  # Check if this is a template (prefixed with __hx_template__:)
  if [[ "$selected" == __hx_template__:* ]]; then
    local tmpl="${selected#__hx_template__:}"
    __hx_expand_template "$tmpl"
  else
    BUFFER="$selected"
    CURSOR=${#BUFFER}
  fi

  zle reset-prompt
  return $ret
}

# Expand a template into the buffer and activate snippet mode
__hx_expand_template() {
  local tmpl="$1"
  local json
  json=$(command hx expand "$tmpl" 2>/dev/null)

  if [[ -z "$json" ]]; then
    # Expansion failed, just use raw template
    BUFFER="$tmpl"
    CURSOR=${#BUFFER}
    return
  fi

  # Parse JSON — extract expanded string and placeholder positions
  # Using zsh parameter expansion to parse simple JSON from hx expand
  local expanded
  expanded="${json#*\"expanded\":\"}"
  expanded="${expanded%%\"*}"
  # Unescape JSON string (handle \", \\, \n, \t)
  expanded="${expanded//\\\"/\"}"
  expanded="${expanded//\\\\/\\}"

  BUFFER="$expanded"

  # Parse placeholders array
  __hx_snippet_starts=()
  __hx_snippet_ends=()
  __hx_snippet_labels=()
  __hx_snippet_index=0
  __hx_snippet_active=0

  local rest="${json#*\"placeholders\":\[}"
  rest="${rest%\]*}"

  while [[ "$rest" == *'"start":'* ]]; do
    local start end label

    # Extract start
    local s="${rest#*\"start\":}"
    start="${s%%[,\}]*}"

    # Extract end
    local e="${rest#*\"end\":}"
    end="${e%%[,\}]*}"

    # Extract label
    local l="${rest#*\"label\":\"}"
    label="${l%%\"*}"

    __hx_snippet_starts+=("$start")
    __hx_snippet_ends+=("$end")
    __hx_snippet_labels+=("$label")

    # Move past this object
    rest="${rest#*\}}"
    # Strip leading comma if present
    rest="${rest#,}"
  done

  if (( ${#__hx_snippet_starts[@]} > 0 )); then
    __hx_snippet_active=1
    __hx_snippet_index=0
    __hx_select_current_placeholder
    # Install snippet keymap
    __hx_install_snippet_keymap
  else
    CURSOR=${#BUFFER}
  fi
}

# Select the current placeholder (set MARK, CURSOR, REGION_ACTIVE)
__hx_select_current_placeholder() {
  local idx=$(( __hx_snippet_index + 1 ))
  if (( idx > ${#__hx_snippet_starts[@]} )); then
    # All done
    __hx_snippet_cleanup
    CURSOR=${#BUFFER}
    return
  fi

  local start="${__hx_snippet_starts[$idx]}"
  local end="${__hx_snippet_ends[$idx]}"

  MARK=$start
  CURSOR=$end
  REGION_ACTIVE=1
}

# Jump to next placeholder
__hx_snippet_next() {
  if (( ! __hx_snippet_active )); then
    return
  fi

  (( __hx_snippet_index++ ))

  if (( __hx_snippet_index >= ${#__hx_snippet_starts[@]} )); then
    # All placeholders visited — clean up and move cursor to end
    __hx_snippet_cleanup
    CURSOR=${#BUFFER}
    REGION_ACTIVE=0
    return
  fi

  __hx_select_current_placeholder
}
zle -N __hx_snippet_next

# Self-insert that replaces active region when in snippet mode
__hx_snippet_self_insert() {
  if (( __hx_snippet_active && REGION_ACTIVE )); then
    local start=$MARK end=$CURSOR
    if (( start > end )); then
      start=$CURSOR
      end=$MARK
    fi

    # Replace region with typed character
    BUFFER="${BUFFER[1,start]}${KEYS}${BUFFER[end+1,-1]}"
    CURSOR=$(( start + ${#KEYS} ))
    REGION_ACTIVE=0

    # Update placeholder positions: the current one changed size
    local old_len=$(( end - start ))
    local new_len=${#KEYS}
    local delta=$(( new_len - old_len ))
    __hx_adjust_positions_after $start $delta

  elif (( __hx_snippet_active )); then
    # Typing additional characters after region was already replaced
    local pos=$CURSOR
    zle .self-insert
    __hx_adjust_positions_after $pos 1

  else
    zle .self-insert
  fi
}
zle -N __hx_snippet_self_insert

# Backward delete that handles active region in snippet mode
__hx_snippet_backward_delete() {
  if (( __hx_snippet_active && REGION_ACTIVE )); then
    local start=$MARK end=$CURSOR
    if (( start > end )); then
      start=$CURSOR
      end=$MARK
    fi

    BUFFER="${BUFFER[1,start]}${BUFFER[end+1,-1]}"
    CURSOR=$start
    REGION_ACTIVE=0

    local old_len=$(( end - start ))
    local delta=$(( -old_len ))
    __hx_adjust_positions_after $start $delta

  elif (( __hx_snippet_active )); then
    # Deleting characters after region was already replaced
    local pos=$CURSOR
    zle .backward-delete-char
    if (( CURSOR < pos )); then
      __hx_adjust_positions_after $CURSOR -1
    fi

  else
    zle .backward-delete-char
  fi
}
zle -N __hx_snippet_backward_delete

# Adjust positions of placeholders whose start is at or after edit_pos.
# Skips the current placeholder being edited. Uses string position
# rather than array index so tab-stop order != string order is handled.
__hx_adjust_positions_after() {
  local edit_pos=$1
  local delta=$2
  local current_idx=$(( __hx_snippet_index + 1 ))

  local i
  for (( i = 1; i <= ${#__hx_snippet_starts[@]}; i++ )); do
    # Skip the placeholder currently being edited
    if (( i == current_idx )); then
      continue
    fi
    if (( __hx_snippet_starts[$i] >= edit_pos )); then
      __hx_snippet_starts[$i]=$(( __hx_snippet_starts[$i] + delta ))
      __hx_snippet_ends[$i]=$(( __hx_snippet_ends[$i] + delta ))
    fi
  done
}

# Install temporary keymap for snippet mode
__hx_install_snippet_keymap() {
  # Bind Ctrl+] to next placeholder
  bindkey '^]' __hx_snippet_next

  # Override all printable chars (space through tilde) to use snippet-aware self-insert
  bindkey -R ' '-'~' __hx_snippet_self_insert

  # Override backspace
  bindkey '^?' __hx_snippet_backward_delete
  bindkey '^H' __hx_snippet_backward_delete
}

# Restore normal keybindings
__hx_snippet_cleanup() {
  __hx_snippet_active=0
  REGION_ACTIVE=0

  # Restore normal self-insert for all printable chars
  bindkey -R ' '-'~' self-insert

  # Restore normal backspace
  bindkey '^?' backward-delete-char
  bindkey '^H' backward-delete-char

  # Remove Ctrl+] binding
  bindkey -r '^]' 2>/dev/null
}

zle -N __hx_search
bindkey -M emacs '^R' __hx_search
bindkey -M vicmd '^R' __hx_search
bindkey -M viins '^R' __hx_search
`
