# multi-track-audio-extraction

将视频放入和 exe相同的目录，根据提示运行即可

工具会默认导出第一条轨道，其他轨道根据json

json格式

[
  {
    'rownum': 1, 
    'typeName': '1'
  }  
  {
    'rownum': 2, 
    'typeName': '2'
  }
]

rownum    音轨id     从0开始计数
typeName   导出的音轨名称   xxx.mp3
