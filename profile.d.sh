# Cosmic Dashboard - runs once per SSH session on fresh login
# Opt out by creating ~/.skiplogin
if [ ! -f ~/.skiplogin ] && [ -t 1 ]; then
    case "$-" in *i*) ;; *) return 0;; esac

    if [ -x /opt/cosmic-dashboard/cosmic-dashboard ]; then
        marker="/tmp/.cosmic_dashboard_${SSH_TTY:-$RANDOM}"

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
