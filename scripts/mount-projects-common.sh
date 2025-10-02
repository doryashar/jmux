#!/usr/bin/env bash

# Mount script for /projects/common NFS share
# Required for jmux shared storage functionality

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Mount configuration
NFS_SERVER="x-filer21-100.xsight.ent"
NFS_PATH="/project_common"
MOUNT_POINT="/projects/common"
USER_ID="87002638"
GROUP_ID="87001524"
MOUNT_OPTIONS="rw,relatime,vers=3,rsize=65536,wsize=65536,namlen=255,hard,nolock,proto=tcp,timeo=600,retrans=2,sec=sys,local_lock=all"

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        echo -e "${RED}Error: This script must be run as root${NC}" >&2
        echo "Please run: sudo $0" >&2
        exit 1
    fi
}

# Check if NFS utilities are installed
check_nfs_utils() {
    if ! command -v mount.nfs &> /dev/null; then
        echo -e "${RED}Error: NFS utilities not found${NC}" >&2
        echo "Please install nfs-utils (RHEL/CentOS) or nfs-common (Ubuntu/Debian)" >&2
        exit 1
    fi
}

# Check if mount point exists
ensure_mount_point() {
    if [[ ! -d "${MOUNT_POINT}" ]]; then
        echo -e "${YELLOW}Creating mount point: ${MOUNT_POINT}${NC}"
        mkdir -p "${MOUNT_POINT}"
    fi
}

# Check if already mounted
is_mounted() {
    mount | grep -q "${MOUNT_POINT}"
}

# Setup jmux directory structure
setup_jmux_directories() {
    echo -e "${BLUE}Setting up jmux directory structure...${NC}"
    
    local jmux_base="${MOUNT_POINT}/work/dory/jmux"
    local dirs=("${jmux_base}" "${jmux_base}/messages" "${jmux_base}/sessions")
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "${dir}" ]]; then
            mkdir -p "${dir}"
            echo -e "${GREEN}✓ Created directory: ${dir}${NC}"
        fi
    done
    
    # Create users.db if it doesn't exist
    local users_file="${jmux_base}/users.db"
    if [[ ! -f "${users_file}" ]]; then
        touch "${users_file}"
        echo -e "${GREEN}✓ Created users database: ${users_file}${NC}"
    fi
    
    # Set permissions for group access
    chmod -R 775 "${jmux_base}" 2>/dev/null || true
    
    echo -e "${GREEN}✓ jmux directory structure ready${NC}"
}

# Mount the NFS share
mount_nfs() {
    if is_mounted; then
        echo -e "${GREEN}✓ ${MOUNT_POINT} is already mounted${NC}"
        mount | grep "${MOUNT_POINT}"
        return 0
    fi
    
    echo -e "${BLUE}Mounting NFS share...${NC}"
    echo "Server: ${NFS_SERVER}:${NFS_PATH}"
    echo "Mount point: ${MOUNT_POINT}"
    echo "User ID: ${USER_ID}"
    echo "Group ID: ${GROUP_ID}"
    echo "Options: ${MOUNT_OPTIONS}"
    echo mount -t nfs -o "${MOUNT_OPTIONS}" "${NFS_SERVER}:${NFS_PATH}" "${MOUNT_POINT}"
    if mount -t nfs -o "${MOUNT_OPTIONS}" "${NFS_SERVER}:${NFS_PATH}" "${MOUNT_POINT}"; then
        echo -e "${GREEN}✓ Successfully mounted ${MOUNT_POINT}${NC}"
        mount | grep "${MOUNT_POINT}"
        
        # Set up jmux directory structure with proper permissions
        setup_jmux_directories
    else
        echo -e "${RED}✗ Failed to mount ${MOUNT_POINT}${NC}" >&2
        exit 1
    fi
}

# Unmount the NFS share
unmount_nfs() {
    if ! is_mounted; then
        echo -e "${YELLOW}${MOUNT_POINT} is not mounted${NC}"
        return 0
    fi
    
    echo -e "${BLUE}Unmounting NFS share...${NC}"
    
    if umount "${MOUNT_POINT}"; then
        echo -e "${GREEN}✓ Successfully unmounted ${MOUNT_POINT}${NC}"
    else
        echo -e "${RED}✗ Failed to unmount ${MOUNT_POINT}${NC}" >&2
        echo -e "${YELLOW}Trying force unmount...${NC}"
        if umount -f "${MOUNT_POINT}"; then
            echo -e "${GREEN}✓ Force unmount successful${NC}"
        else
            echo -e "${RED}✗ Force unmount failed${NC}" >&2
            exit 1
        fi
    fi
}

# Show mount status
show_status() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}NFS Mount Status${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    if is_mounted; then
        echo -e "${GREEN}✓ ${MOUNT_POINT} is mounted${NC}"
        mount | grep "${MOUNT_POINT}"
        echo
        echo -e "${BLUE}Disk usage:${NC}"
        df -h "${MOUNT_POINT}" 2>/dev/null || echo "Unable to get disk usage"
    else
        echo -e "${YELLOW}✗ ${MOUNT_POINT} is not mounted${NC}"
    fi
    
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Add to /etc/fstab for permanent mounting
add_to_fstab() {
    local fstab_entry="${NFS_SERVER}:${NFS_PATH} ${MOUNT_POINT} nfs ${MOUNT_OPTIONS} 0 0"
    
    if grep -q "${MOUNT_POINT}" /etc/fstab; then
        echo -e "${YELLOW}Entry for ${MOUNT_POINT} already exists in /etc/fstab${NC}"
        grep "${MOUNT_POINT}" /etc/fstab
    else
        echo -e "${BLUE}Adding entry to /etc/fstab for automatic mounting...${NC}"
        echo "${fstab_entry}" >> /etc/fstab
        echo -e "${GREEN}✓ Added to /etc/fstab${NC}"
        echo "${fstab_entry}"
    fi
}

# Remove from /etc/fstab
remove_from_fstab() {
    if grep -q "${MOUNT_POINT}" /etc/fstab; then
        echo -e "${BLUE}Removing entry from /etc/fstab...${NC}"
        sed -i "\|${MOUNT_POINT}|d" /etc/fstab
        echo -e "${GREEN}✓ Removed from /etc/fstab${NC}"
    else
        echo -e "${YELLOW}No entry found in /etc/fstab for ${MOUNT_POINT}${NC}"
    fi
}

# Setup autofs configuration
setup_autofs() {
    local auto_master="/etc/auto.master"
    local auto_direct="/etc/auto.master.d/auto.direct"
    
    echo -e "${BLUE}Setting up autofs configuration...${NC}"
    
    # Ensure autofs is installed
    if ! command -v automount &> /dev/null; then
        echo -e "${RED}Error: autofs not installed${NC}" >&2
        echo "Please install autofs package" >&2
        exit 1
    fi
    
    # Create auto.direct file
    mkdir -p /etc/auto.master.d
    echo "${MOUNT_POINT} -${MOUNT_OPTIONS} ${NFS_SERVER}:${NFS_PATH}" > "${auto_direct}"
    
    # Add direct map to auto.master if not present
    if ! grep -q "/etc/auto.master.d/auto.direct" "${auto_master}"; then
        echo "/- /etc/auto.master.d/auto.direct" >> "${auto_master}"
    fi
    
    echo -e "${GREEN}✓ Autofs configuration created${NC}"
    echo -e "${BLUE}Restarting autofs service...${NC}"
    
    systemctl restart autofs
    systemctl enable autofs
    
    echo -e "${GREEN}✓ Autofs service restarted and enabled${NC}"
}

# Show help
show_help() {
    cat << 'EOF'
Mount script for /projects/common NFS share

USAGE:
    sudo ./mount-projects-common.sh [command]

COMMANDS:
    mount       Mount the NFS share (default)
    unmount     Unmount the NFS share
    status      Show current mount status
    fstab       Add entry to /etc/fstab for permanent mounting
    remove-fstab Remove entry from /etc/fstab
    autofs      Setup autofs configuration
    setup-dirs  Setup jmux directory structure (requires mount to exist)
    help        Show this help message

EXAMPLES:
    # Mount the share
    sudo ./mount-projects-common.sh mount

    # Check status
    sudo ./mount-projects-common.sh status

    # Add to fstab for automatic mounting at boot
    sudo ./mount-projects-common.sh fstab

    # Setup autofs (recommended)
    sudo ./mount-projects-common.sh autofs

    # Setup jmux directories after mounting
    sudo ./mount-projects-common.sh setup-dirs

NOTES:
    - Requires root privileges
    - NFS utilities must be installed
    - Server: x-filer21-100.xsight.ent:/project_common
    - Mount point: /projects/common
    - User ID: 87002638, Group ID: 87001524 (for reference only - NFS handles ownership)
    - This mount is required for jmux shared storage functionality
    - Directory structure: /projects/common/work/dory/jmux/

EOF
}

# Main function
main() {
    local cmd="${1:-mount}"
    
    case "${cmd}" in
        mount)
            check_root
            check_nfs_utils
            ensure_mount_point
            mount_nfs
            ;;
        unmount)
            check_root
            unmount_nfs
            ;;
        status)
            show_status
            ;;
        fstab)
            check_root
            ensure_mount_point
            add_to_fstab
            ;;
        remove-fstab)
            check_root
            remove_from_fstab
            ;;
        autofs)
            check_root
            setup_autofs
            ;;
        setup-dirs)
            check_root
            if ! is_mounted; then
                echo -e "${RED}Error: ${MOUNT_POINT} is not mounted${NC}" >&2
                echo "Please mount the NFS share first" >&2
                exit 1
            fi
            setup_jmux_directories
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}Error: Unknown command '${cmd}'${NC}" >&2
            echo "Run './mount-projects-common.sh help' for usage information" >&2
            exit 1
            ;;
    esac
}

main "$@"
