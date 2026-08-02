# Cosmic Dashboard - runs once per SSH session on fresh login
# Opt out by creating ~/.skiplogin
if [ ! -f ~/.skiplogin ] && [ -n "$SSH_TTY" ] && [ -t 1 ] && [ -x /opt/cosmic-dashboard/cosmic-dashboard ]; then
    case "$-" in *i*) ;; *) return 0;; esac
    marker="/tmp/.cosmic_dashboard_${SSH_TTY//\//_}"
    # Cleanup: remove markers whose PTY no longer exists
    for m in /tmp/.cosmic_dashboard__dev_pts_*; do
        [ -f "$m" ] || continue
        pty="/dev/pts/${m##*_}"
        [ -e "$pty" ] || rm -f "$m"
    done
    if [ ! -f "$marker" ]; then
        /opt/cosmic-dashboard/cosmic-dashboard
        touch "$marker"
    fi
fi
