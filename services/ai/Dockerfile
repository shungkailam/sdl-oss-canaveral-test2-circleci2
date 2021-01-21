FROM docker.dyn.ntnxdpro.com/ntnx-general-docker/python:3.7
ENV GRPC_PYTHON_VERSION 1.21.1
RUN python -m pip install --upgrade setuptools
RUN python -m pip install --upgrade pip
RUN pip install grpcio==${GRPC_PYTHON_VERSION} grpcio-tools==${GRPC_PYTHON_VERSION}
RUN mkdir /ai
COPY ./ai /ai/
WORKDIR /ai
RUN pip install -r /ai/requirements.txt
RUN python -m grpc_tools.protoc -I./ --python_out=. --grpc_python_out=. ./proto/*.proto
EXPOSE 8500
CMD ["python", "server.py"]
