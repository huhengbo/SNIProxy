#!/bin/bash

# 设置版本号
version="v1.0.5"

# 打包的输出目录
output_dir="Releases"

# 定义所有的目标平台和架构
# 格式为：操作系统-架构
platforms=("linux-amd64" "linux-arm64" "windows-amd64" "darwin-amd64" "darwin-arm64")

# 遍历所有平台进行打包
for platform in "${platforms[@]}"; do
    # 分割操作系统和架构
    IFS="-" read -r GOOS GOARCH <<< "$platform"

    # 设置输出文件夹和文件名路径
    output_folder="${output_dir}/sniproxy_${GOOS}_${GOARCH}"
    output_file="${output_folder}/sniproxy"

    # 针对Windows的输出文件需要带.exe后缀
    if [ "$GOOS" = "windows" ]; then
        output_file="${output_file}.exe"
    fi

    # 创建目标目录（如果不存在）
    mkdir -p "$output_folder"

    # 打包
    echo "正在打包 $GOOS $GOARCH ..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$output_file" -ldflags "-s -w -X main.version=$version"

    # 检查打包是否成功
    if [ $? -ne 0 ]; then
        echo "打包 $GOOS $GOARCH 失败！"
        exit 1
    fi

    # 复制 config.yaml 文件到目标目录
    if [ -f "config.yaml" ]; then
        cp config.yaml "$output_folder/"
    else
        echo "未找到 config.yaml 文件，跳过复制。"
    fi

    # 打包完成，开始压缩文件
    if [ "$GOOS" = "linux" ]; then
        # 对于Linux，使用 tar.gz 格式
        tar -czvf "${output_folder}.tar.gz" -C "$output_folder" .
        if [ $? -ne 0 ]; then
            echo "压缩 $GOOS $GOARCH 失败！"
            exit 1
        fi
        # 删除临时文件夹
        rm -rf "$output_folder"
    else
        # 对于Windows和macOS，使用 zip 格式
        zip -r "${output_folder}.zip" "$output_folder"
        if [ $? -ne 0 ]; then
            echo "压缩 $GOOS $GOARCH 失败！"
            exit 1
        fi
        # 删除临时文件夹
        rm -rf "$output_folder"
    fi

    echo "打包并压缩 $GOOS $GOARCH 成功！"
done

echo "所有平台打包并压缩完成！"
