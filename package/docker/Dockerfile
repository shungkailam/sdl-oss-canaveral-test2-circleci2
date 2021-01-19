#alpine is a package which is much smaller. You can also choose ubutu as basic image by change it to FROM ubuntu:latest
FROM alpine
MAINTAINER Xi Eng Ops <canaveral@internal.nutanix.com>

EXPOSE 80

# Update
# RUN apk add --update python py2-pip

# Bundle app source
COPY hello-world /opt/hello-world

# Install requirement
#RUN pip install -U pip wheel &\
#    pip install -r /opt/hello-world/requirements.txt

CMD ["python", "/opt/hello-world/code/controller.py"]
