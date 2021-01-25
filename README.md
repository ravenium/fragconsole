# fragconsole
Quick start:

docker build -t srtrelay -f Dockerfile.srtrelay .

docker build -t fragconsole -f Dockerfile.fragconsole .

docker-compose up

Push your favorite SRT streams to srt://your-docker-ip:1935?streamid=publish/yourcustomstreamid

View streams at http://your-docker-ip:3000/monitor