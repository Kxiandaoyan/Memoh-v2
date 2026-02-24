#!/bin/bash
# migrate-tools-md.sh
# 一次性迁移脚本：为所有现有bots更新TOOLS.md到最新版本
#
# 用法:
#   ./scripts/migrate-tools-md.sh [DATA_ROOT]
#
# 参数:
#   DATA_ROOT  数据根目录,默认为 ./data
#
# 功能:
#   - 查找所有已创建的bot目录 (DATA_ROOT/bots/*)
#   - 将最新的 cmd/mcp/template/TOOLS.md 复制到每个bot的 /TOOLS.md
#   - 强制覆盖现有文件 (cp -f)
#   - 记录每个成功更新的bot ID
#
# 注意:
#   - 这会覆盖bot的现有TOOLS.md文件
#   - 如果bot有自定义内容,建议先备份
#   - 运行前请确保 cmd/mcp/template/TOOLS.md 是最新版本

set -e

# 配置
DATA_ROOT=${1:-./data}
TEMPLATE_FILE=cmd/mcp/template/TOOLS.md

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查模板文件
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo -e "${RED}Error: Template file not found: $TEMPLATE_FILE${NC}"
    exit 1
fi

# 检查数据目录
if [ ! -d "$DATA_ROOT/bots" ]; then
    echo -e "${YELLOW}Warning: No bots directory found at $DATA_ROOT/bots${NC}"
    echo "Creating directory..."
    mkdir -p "$DATA_ROOT/bots"
    echo -e "${GREEN}✓ Directory created${NC}"
    exit 0
fi

# 获取模板版本
TEMPLATE_VERSION=$(head -n 1 "$TEMPLATE_FILE" | grep -o 'version: [^-]*' | cut -d' ' -f2 || echo "unknown")

echo "================================================"
echo "TOOLS.md Migration Script"
echo "================================================"
echo "Template: $TEMPLATE_FILE"
echo "Version:  $TEMPLATE_VERSION"
echo "Data root: $DATA_ROOT"
echo "================================================"
echo ""

# 统计
total_bots=0
updated_bots=0
failed_bots=0

# 遍历所有bot目录
for bot_dir in "$DATA_ROOT"/bots/*; do
    if [ -d "$bot_dir" ]; then
        bot_id=$(basename "$bot_dir")
        total_bots=$((total_bots + 1))

        # 检查是否已存在TOOLS.md
        if [ -f "$bot_dir/TOOLS.md" ]; then
            # 读取现有版本
            old_version=$(head -n 1 "$bot_dir/TOOLS.md" | grep -o 'version: [^-]*' | cut -d' ' -f2 || echo "none")
            echo -e "${YELLOW}[$bot_id]${NC} Current version: $old_version → $TEMPLATE_VERSION"
        else
            echo -e "${YELLOW}[$bot_id]${NC} No existing TOOLS.md, creating new file"
        fi

        # 复制模板文件 (强制覆盖)
        if cp -f "$TEMPLATE_FILE" "$bot_dir/TOOLS.md"; then
            echo -e "${GREEN}  ✓ Updated successfully${NC}"
            updated_bots=$((updated_bots + 1))
        else
            echo -e "${RED}  ✗ Update failed${NC}"
            failed_bots=$((failed_bots + 1))
        fi
        echo ""
    fi
done

# 打印统计
echo "================================================"
echo "Migration Summary"
echo "================================================"
echo "Total bots:   $total_bots"
echo -e "Updated:      ${GREEN}$updated_bots${NC}"
echo -e "Failed:       ${RED}$failed_bots${NC}"
echo "================================================"

if [ $failed_bots -gt 0 ]; then
    echo -e "${RED}⚠ Some bots failed to update. Please check manually.${NC}"
    exit 1
else
    echo -e "${GREEN}✓ All bots updated successfully!${NC}"
    exit 0
fi
