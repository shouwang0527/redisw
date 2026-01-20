# 图标说明

浏览器插件需要 PNG 格式的图标。请使用以下命令将 SVG 转换为 PNG：

```bash
# 使用 ImageMagick 转换
convert icon16.svg icon16.png
convert icon48.svg icon48.png
convert icon128.svg icon128.png

# 或使用 rsvg-convert
rsvg-convert -w 16 -h 16 icon16.svg > icon16.png
rsvg-convert -w 48 -h 48 icon48.svg > icon48.png
rsvg-convert -w 128 -h 128 icon128.svg > icon128.png
```

或者使用在线工具如 https://cloudconvert.com/svg-to-png 进行转换。
