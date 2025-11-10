# 编写人民日报的pdf爬虫+pdf合并

技术选型：使用golang 进行爬取
resty这个框架

爬取流程：
1. 获取当前日期，今天的时间，生成人民日报首版爬取的地址
我举个例子：`https://paper.people.com.cn/rmrb/pc/layout/202511/10/node_01.html`
其中202511/10,就是表示的年月日，node_01.html就是第一版的信息，如果是第二版，则是node_02

2. 抓取其中的版数：
body > div.main.w1000 > div.right.right-main > div.swiper-box > div > div:nth-child(1)

3. 抓取当前版面的pdf文件
body > div.main.w1000 > div.left.paper-box > div.paper-bot > p.right.btn > a

4. 将下载的pdf文件放到 `web/files`目录下，进行编号，要跟上面的一致

5. 将下载的pdf合并，然后把合并结果的pdf文件放到`dist`目录下

