# Cosmic Dashboard - runs once per SSH session on fresh login
# Opt out by creating ~/.skiplogin
if [ ! -f ~/.skiplogin ]; then
    case "$-" in *i*) ;; *) return 0;; esac

    if [ -x /opt/cosmic-dashboard/cosmic-dashboard ]; then
        marker="/tmp/.cosmic_dashboard_${SSH_TTY//\//_}"

        # Cleanup stale markers: remove any whose PTY no longer exists
        for m in /tmp/.cosmic_dashboard__dev_pts_*; do
            [ -f "$m" ] || continue
            pty="/dev/pts/${m##*_}"
            [ -e "$pty" ] || rm -f "$m"
        done

        if [ -f "$marker" ]; then
            marker_age=$(( $(date +%s) - $(stat -c %Y "$marker" 2>/dev/null || echo 0) ))
            if [ "$marker_age" -lt 3600 ]; then
                return 0
            fi
        fi

        exec 9>"$marker"
        if ! flock -n 9; then
            return 0
        fi

        timeout 8s /opt/cosmic-dashboard/cosmic-dashboard || true

        touch "$marker"
    fi
fi
