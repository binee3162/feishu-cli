package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/riba2534/feishu-cli/internal/client"
	"github.com/riba2534/feishu-cli/internal/config"
	"github.com/spf13/cobra"
)

var imImageUploadCmd = &cobra.Command{
	Use:   "im-image-upload <file>",
	Short: "上传图片到飞书（IM 图片）",
	Long: `通过 IM API 上传图片，获取可在消息中使用的 image_key。

与 media upload 不同，此命令使用 IM 图片接口（/open-apis/im/v1/images），
上传后的图片可直接用于发送图片消息、富文本消息或卡片消息。

示例:
  # 上传图片获取 image_key
  feishu-cli media im-image-upload screenshot.png
  # 输出: image_key: img_v2_xxx

  # JSON 格式输出（方便脚本使用）
  feishu-cli media im-image-upload photo.jpg -o json
  # 输出: {"image_key":"img_v2_xxx"}`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Validate(); err != nil {
			return err
		}

		filePath := args[0]
		imageKey, err := client.UploadIMImage(filePath)
		if err != nil {
			return err
		}

		output, _ := cmd.Flags().GetString("output")
		if output == "json" {
			result, _ := json.Marshal(map[string]string{
				"image_key": imageKey,
			})
			fmt.Println(string(result))
		} else {
			fmt.Printf("image_key: %s\n", imageKey)
		}
		return nil
	},
}

func init() {
	mediaCmd.AddCommand(imImageUploadCmd)
	imImageUploadCmd.Flags().StringP("output", "o", "", "输出格式（json）")
}
