package cmd

import (
	"fmt"

	"github.com/riba2534/feishu-cli/internal/client"
	"github.com/riba2534/feishu-cli/internal/config"
	"github.com/spf13/cobra"
)

var uploadImageCmd = &cobra.Command{
	Use:   "image-upload <file>",
	Short: "上传图片到飞书（IM 图片）",
	Long: `通过 IM API 上传本地图片，获取 image_key。

返回的 image_key 可用于：
- 发送图片消息：--msg-type image --content '{"image_key":"img_xxx"}'
- 富文本消息中内嵌图片：{"tag":"img","image_key":"img_xxx"}
- 卡片消息中内嵌图片：{"tag":"img","img_key":"img_xxx","alt":{"tag":"plain_text","content":"描述"}}

支持格式：JPEG、PNG、BMP、GIF、TIFF、WebP
大小限制：10MB

示例:
  # 上传图片
  feishu-cli msg image-upload screenshot.png

  # JSON 格式输出（方便脚本调用）
  feishu-cli msg image-upload photo.jpg -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Validate(); err != nil {
			return err
		}

		filePath := args[0]

		fmt.Fprintf(cmd.ErrOrStderr(), "正在上传图片: %s\n", filePath)
		imageKey, err := client.UploadIMImage(filePath)
		if err != nil {
			return err
		}

		output, _ := cmd.Flags().GetString("output")
		if output == "json" {
			if err := printJSON(map[string]string{
				"image_key": imageKey,
			}); err != nil {
				return err
			}
		} else {
			fmt.Printf("图片上传成功！\n")
			fmt.Printf("  image_key: %s\n", imageKey)
		}

		return nil
	},
}

func init() {
	msgCmd.AddCommand(uploadImageCmd)
	uploadImageCmd.Flags().StringP("output", "o", "", "输出格式（json）")
}
