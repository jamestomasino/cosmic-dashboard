# Cosmic Dashboard - runs once per SSH session on fresh login
# Opt out by creating ~/.skiplogin
if [ ! -f ~/.skiplogin ] && [ -t 1 ] && [ -x /opt/cosmic-dashboard/cosmic-dashboard ]; then
    case "$-" in *i*) ;; *) return 0;; esac
    # Skip if current PTY is not the original SSH PTY (tmux panes)
    [ "$(tty)" = "$SSH_TTY" ] || return 0
    marker="/tmp/.cosmic_dashboard_${SSH_TTY//\//_}"
    if [ -f "$marker" ]; then
        marker_age=$(( $(date +%s) - $(stat -c %Y "$marker" 2>/dev/null || echo 0) ))
        [ "$marker_age" -lt 30 ] && return 0
    fi
    /opt/cosmic-dashboard/cosmic-dashboard
    touch "$marker"
fi
