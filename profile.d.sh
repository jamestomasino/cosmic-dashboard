# Cosmic Dashboard - runs once per SSH session on fresh login
# Opt out by creating ~/.skiplogin
if [ ! -f ~/.skiplogin ] && [ -n "$SSH_TTY" ] && [ -t 1 ] && [ -x /opt/cosmic-dashboard/cosmic-dashboard ]; then
    case "$-" in *i*) ;; *) return 0;; esac
    marker="/tmp/.cosmic_dashboard_${SSH_TTY//\//_}"
    # Cleanup: remove markers whose PTY no longer exists
    for m in /tmp/.cosmic_dashboard__dev_pts_*; do
        [ -f "$m" ] || continue
        pty="/dev/pts/${m##*_}"
        [ -e "$pty" ] || rm -f "$m" 2>/dev/null
    done
    # TTL: remove marker if older than 30s (previous session)
    if [ -f "$marker" ]; then
        marker_age=$(( $(date +%s) - $(stat -c %Y "$marker" 2>/dev/null || echo 0) ))
        [ "$marker_age" -lt 30 ] && return 0
    fi
    /opt/cosmic-dashboard/cosmic-dashboard
    touch "$marker"
fi
