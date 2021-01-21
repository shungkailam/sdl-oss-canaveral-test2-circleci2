package config

import (
	"cloudservices/common/model"
)

// BuiltinCategories are the builtin categories
var BuiltinCategories = []model.Category{
	model.Category{
		BaseModel: model.BaseModel{
			ID:      "cat-data-type",
			Version: 0,
		},
		Name:    "Data Type",
		Purpose: "To specify data type for each field in a data source.",
		Values: []string{
			"Custom",
			"Humidity",
			"Image",
			"Light",
			"Motion",
			"Pressure",
			"Processed",
			"Proximity",
			"Temperature",
		},
	},
}

// BuiltinProjects are the builtin projects
var BuiltinProjects = []model.Project{
	model.Project{
		Name:             "Default Project",
		Description:      "Default Project for backward compatibility",
		EdgeSelectorType: "Explicit",
	},
}

// BuiltinScriptRuntimes are the builtin script runtimes
var BuiltinScriptRuntimes = []model.ScriptRuntime{
	model.ScriptRuntime{
		BaseModel: model.BaseModel{ID: "sr-python",
			Version: 0,
		},
		ScriptRuntimeCore: model.ScriptRuntimeCore{
			Name:          "Python3 Env",
			Description:   "Python 3 Runtime",
			Builtin:       true,
			Language:      "python",
			DockerRepoURI: "python-env",
			Dockerfile: `FROM python:3.6

COPY ./runtimes/python-env/build/python-env.tgz /
RUN tar xf /python-env.tgz
RUN pip install -r /python-env/requirements.txt

CMD ["/python-env/run.sh"]`,
		},
	},
	model.ScriptRuntime{
		BaseModel: model.BaseModel{ID: "sr-python2",
			Version: 0,
		},
		ScriptRuntimeCore: model.ScriptRuntimeCore{
			Name:          "Python2 Env",
			Description:   "Python 2 Runtime",
			Builtin:       true,
			Language:      "python",
			DockerRepoURI: "python2-env",
			Dockerfile: `FROM python:2.7

COPY ./runtimes/python2-env/build/python-env.tgz /
RUN tar xf /python-env.tgz
RUN pip install -r /python-env/requirements.txt

CMD ["/python-env/run.sh"]`,
		},
	},
	model.ScriptRuntime{
		BaseModel: model.BaseModel{ID: "sr-tensorflow",
			Version: 0,
		},
		ScriptRuntimeCore: model.ScriptRuntimeCore{
			Name:          "Tensorflow Python",
			Description:   "Tensorflow Python Runtime",
			Builtin:       true,
			Language:      "python",
			DockerRepoURI: "tensorflow-python",
			Dockerfile: `FROM tensorflow/tensorflow:1.7.0

RUN apt-get update
RUN apt-get install -y wget
RUN apt-get install -y python-opencv
RUN apt-get install -y vim

#Install Python2 FaaS runtime
COPY ./runtimes/python2-env/build/python-env.tgz /
WORKDIR /
RUN tar xf /python-env.tgz
RUN pip install -r /python-env/requirements.txt

#Copy object detection files
RUN mkdir /mllib
RUN mkdir /mllib/objectdetection
WORKDIR /mllib/objectdetection
COPY ./runtimes/tensorflow-python/objectdetection/  /mllib/objectdetection/

#Download tensorflow object detection model
RUN curl -O http://download.tensorflow.org/models/object_detection/ssd_inception_v2_coco_2017_11_17.tar.gz
RUN tar -xvzf ssd_inception_v2_coco_2017_11_17.tar.gz
RUN rm -rf ssd_inception_v2_coco_2017_11_17.tar.gz

#Copy face recognition files
RUN mkdir -p /mllib/facerecognition/20180402-114759
COPY ./runtimes/tensorflow-python/facerecognition/  /mllib/facerecognition/
WORKDIR /mllib/facerecognition

#Download tensorflow facenet recognition model from s3
RUN wget -O /mllib/facerecognition/20180402-114759/20180402-114759.pb "https://sherlock-facenet-model.s3.amazonaws.com/20180402-114759.pb?AWSAccessKeyId=AKIAIGEED27MJSXDY5DQ&Expires=1681341750&Signature=8Pz%2BGk0ntmfXIJiyUDyJfJ%2FfpBM%3D"
RUN pip install -r requirements.txt

ENV PYTHONPATH="/mllib:${PYTHONPATH}"

CMD ["/python-env/run.sh"]`,
		},
	},
	model.ScriptRuntime{
		BaseModel: model.BaseModel{ID: "sr-node",
			Version: 0,
		},
		ScriptRuntimeCore: model.ScriptRuntimeCore{
			Name:          "Node Env",
			Description:   "NodeJS Runtime",
			Builtin:       true,
			Language:      "node",
			DockerRepoURI: "node-env",
			Dockerfile: `FROM node:9-alpine

COPY ./runtimes/node-env/build/node-env.tgz /

RUN tar xf /node-env.tgz

WORKDIR /node-env

RUN npm install

CMD ["/node-env/run.sh"]`,
		},
	},
	model.ScriptRuntime{
		BaseModel: model.BaseModel{ID: "sr-go",
			Version: 0,
		},
		ScriptRuntimeCore: model.ScriptRuntimeCore{
			Name:          "Golang Env",
			Description:   "Golang Runtime",
			Builtin:       true,
			Language:      "golang",
			DockerRepoURI: "golang-env",
			Dockerfile: `FROM golang:1.9

RUN go env
RUN mkdir -p /go/src/main
RUN mkdir -p /go/src/nutanix.com/sherlock
COPY ./runtimes/golang_env/ /go/src/nutanix.com/sherlock/runtime
RUN mkdir -p /go/src/nutanix.com/sherlock/runtime/datastream
COPY ./generated/proto/datastream/ /go/src/nutanix.com/sherlock/runtime/datastream
COPY ./runtimes/bootstrap/bootstrap /go/src/nutanix.com/sherlock/runtime

# For testing
RUN go build nutanix.com/sherlock/runtime/test

WORKDIR /go/src/nutanix.com/sherlock/runtime/
CMD ["./run.sh"]`,
		},
	},
}
