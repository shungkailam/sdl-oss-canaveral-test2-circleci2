import { getSha256 } from '../util/cryptoUtil';
import * as uuidv4 from 'uuid/v4';
import platformService from '../services/platform.service';

// unique per tenant
export const TENANT_TOKEN = 'token1'; // genTenantToken();
export const TENANT_TOKEN_2 = 'token2'; //genTenantToken();
export const TENANT_ID = 'tenant-id-waldot';
export const TENANT_ID_2 = 'tenant_id_rocket_blue';
import {
  CloudType,
  DataStream,
  EdgeStreamType,
  DataStreamDestination,
  GCPStreamType,
  AWSStreamType,
  AWS_REGION,
  GCP_REGION,
} from '../model/index';

const USER_1 = {
  id: uuidv4(),
  name: 'Demo',
  email: 'demo@nutanix.com',
  password: getSha256('apex'),
  tenantId: TENANT_ID,
  role: 'INFRA_ADMIN',
};
const USER_2 = {
  id: uuidv4(),
  name: 'Demo 2',
  email: 'demo2@nutanix.com',
  password: getSha256('apex'),
  tenantId: TENANT_ID_2,
  role: 'INFRA_ADMIN',
};
export function getUser(tenantId) {
  switch (tenantId) {
    case TENANT_ID:
      return USER_1;
    case TENANT_ID_2:
      return USER_2;
    default:
      return null;
  }
}

export const SCRIPT_RUNTIMES = [
  {
    id: 'sr-python',
    name: 'Python3 Env',
    version: 0,
    tenant_id: '',
    description: 'Python 3 Runtime',
    builtin: true,
    language: 'python',
    docker_repo_uri: 'python-env',
    docker_profile_id: '',
    dockerfile: `\
FROM python:3.6

COPY ./runtimes/python-env/build/python-env.tgz /
RUN tar xf /python-env.tgz
RUN pip install -r /python-env/requirements.txt

CMD ["/python-env/run.sh"]`,
    created_at: '2018-01-01T01:01:01Z',
    updated_at: '2018-01-01T01:01:01Z',
  },
  {
    id: 'sr-python2',
    name: 'Python2 Env',
    version: 0,
    tenant_id: '',
    description: 'Python 2 Runtime',
    builtin: true,
    language: 'python',
    docker_repo_uri: 'python2-env',
    docker_profile_id: '',
    dockerfile: `\
FROM python:2.7

COPY ./runtimes/python2-env/build/python-env.tgz /
RUN tar xf /python-env.tgz
RUN pip install -r /python-env/requirements.txt

CMD ["/python-env/run.sh"]`,
    created_at: '2018-01-01T01:01:01Z',
    updated_at: '2018-01-01T01:01:01Z',
  },
  {
    id: 'sr-tensorflow',
    name: 'Tensorflow Python',
    version: 0,
    tenant_id: '',
    description: 'Tensorflow Python Runtime',
    builtin: true,
    language: 'python',
    docker_repo_uri: 'tensorflow-python',
    docker_profile_id: '',
    dockerfile: `\
FROM tensorflow/tensorflow:1.7.0

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

ENV PYTHONPATH="/mllib:\${PYTHONPATH}"

CMD ["/python-env/run.sh"]`,
    created_at: '2018-01-01T01:01:01Z',
    updated_at: '2018-01-01T01:01:01Z',
  },
  {
    id: 'sr-node',
    name: 'Node Env',
    version: 0,
    tenant_id: '',
    description: 'NodeJS Runtime',
    builtin: true,
    language: 'node',
    docker_repo_uri: 'node-env',
    docker_profile_id: '',
    dockerfile: `\
FROM node:9-alpine

COPY ./runtimes/node-env/build/node-env.tgz /

RUN tar xf /node-env.tgz

WORKDIR /node-env

RUN npm install

CMD ["/node-env/run.sh"]`,
    created_at: '2018-01-01T01:01:01Z',
    updated_at: '2018-01-01T01:01:01Z',
  },
  {
    id: 'sr-go',
    name: 'Golang Env',
    version: 0,
    tenant_id: '',
    description: 'Golang Runtime',
    builtin: true,
    language: 'golang',
    docker_repo_uri: 'golang-env',
    docker_profile_id: '',
    dockerfile: `\
FROM golang:1.9

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
    created_at: '2018-01-01T01:01:01Z',
    updated_at: '2018-01-01T01:01:01Z',
  },
];

export const APP_STATUS = {
  version: 844778999,
  tenantId: 'tenant-id-nikita',
  edgeId: 'e9aadf87-f2f0-4083-a75b-74f4861a2408',
  applicationId: '66e655a5-1564-48c4-8ab1-84f05b70615b',
  createdAt: '2018-06-27T23:51:54.897794Z',
  updatedAt: '2018-06-28T11:57:54.844779Z',
  appStatus: {
    podStatusList: [
      {
        metadata: {
          annotations: {
            'kubernetes.io/created-by':
              '{"kind":"SerializedReference","apiVersion":"v1","reference":{"kind":"ReplicaSet","namespace":"66e655a5-1564-48c4-8ab1-84f05b70615b","name":"facefeed-deployment-64bcd6dcc5","uid":"ee2fad1d-7a64-11e8-9219-506b8db6fe5b","apiVersion":"extensions","resourceVersion":"1067"}}\n',
          },
          creationTimestamp: '2018-06-27T23:50:54Z',
          generateName: 'facefeed-deployment-64bcd6dcc5-',
          labels: {
            app: 'facefeed',
            'pod-template-hash': '2067828771',
          },
          name: 'facefeed-deployment-64bcd6dcc5-5562d',
          namespace: '66e655a5-1564-48c4-8ab1-84f05b70615b',
          ownerReferences: [
            {
              apiVersion: 'extensions/v1beta1',
              blockOwnerDeletion: true,
              controller: true,
              kind: 'ReplicaSet',
              name: 'facefeed-deployment-64bcd6dcc5',
              uid: 'ee2fad1d-7a64-11e8-9219-506b8db6fe5b',
            },
          ],
          resourceVersion: '38995',
          selfLink:
            '/api/v1/namespaces/66e655a5-1564-48c4-8ab1-84f05b70615b/pods/facefeed-deployment-64bcd6dcc5-5562d',
          uid: 'ee3ddb0a-7a64-11e8-9219-506b8db6fe5b',
        },
        spec: {
          containers: [
            {
              command: ['sh', '-c', 'exec python main.py'],
              image:
                '770301640873.dkr.ecr.us-west-2.amazonaws.com/face-feed-app:v1',
              imagePullPolicy: 'Always',
              name: 'facefeed',
              ports: [
                {
                  containerPort: 8888,
                  protocol: 'TCP',
                },
              ],
              resources: {},
              terminationMessagePath: '/dev/termination-log',
              terminationMessagePolicy: 'File',
              volumeMounts: [
                {
                  mountPath: '/var/run/secrets/kubernetes.io/serviceaccount',
                  name: 'default-token-w6spd',
                  readOnly: true,
                },
              ],
            },
          ],
          dnsPolicy: 'ClusterFirst',
          imagePullSecrets: [
            {
              name: 'face-feed',
            },
            {
              name: 'flask-web-hub',
            },
            {
              name: 'flaskweb',
            },
          ],
          nodeName: 'nos-nt-1',
          restartPolicy: 'Always',
          schedulerName: 'default-scheduler',
          securityContext: {},
          serviceAccount: 'default',
          serviceAccountName: 'default',
          terminationGracePeriodSeconds: 30,
          tolerations: [
            {
              effect: 'NoExecute',
              key: 'node.alpha.kubernetes.io/notReady',
              operator: 'Exists',
              tolerationSeconds: 300,
            },
            {
              effect: 'NoExecute',
              key: 'node.alpha.kubernetes.io/unreachable',
              operator: 'Exists',
              tolerationSeconds: 300,
            },
          ],
          volumes: [
            {
              name: 'default-token-w6spd',
              secret: {
                defaultMode: 420,
                secretName: 'default-token-w6spd',
              },
            },
          ],
        },
        status: {
          conditions: [
            {
              lastProbeTime: null,
              lastTransitionTime: '2018-06-27T23:50:54Z',
              status: 'True',
              type: 'Initialized',
            },
            {
              lastProbeTime: null,
              lastTransitionTime: '2018-06-27T23:53:56Z',
              status: 'True',
              type: 'Ready',
            },
            {
              lastProbeTime: null,
              lastTransitionTime: '2018-06-27T23:50:54Z',
              status: 'True',
              type: 'PodScheduled',
            },
          ],
          containerStatuses: [
            {
              containerID:
                'docker://c14a882d0f808e9e83cb83768343d058ec9ea7ec7c76c189736ad12e42dea2bc',
              image:
                '770301640873.dkr.ecr.us-west-2.amazonaws.com/face-feed-app:v1',
              imageID:
                'docker-pullable://770301640873.dkr.ecr.us-west-2.amazonaws.com/face-feed-app@sha256:1ac88cfde62fa454ea4b15b396c6729660e4d1648592d9d9b16752681569b85e',
              lastState: {},
              name: 'facefeed',
              ready: true,
              restartCount: 0,
              state: {
                running: {
                  startedAt: '2018-06-27T23:53:56Z',
                },
              },
            },
          ],
          hostIP: '10.15.232.22',
          phase: 'Running',
          podIP: '10.32.0.22',
          qosClass: 'BestEffort',
          startTime: '2018-06-27T23:50:54Z',
        },
      },
      {
        metadata: {
          annotations: {
            'kubectl.kubernetes.io/last-applied-configuration':
              '{"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{},"name":"facefeed-proxy","namespace":"66e655a5-1564-48c4-8ab1-84f05b70615b"},"spec":{"containers":[{"args":["tcp","8888","facefeed-svc"],"image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/gcr_facefeed:latest","name":"facefeed-tcp","ports":[{"containerPort":8888,"hostPort":8888,"name":"tcp","protocol":"TCP"}]}]}}\n',
          },
          creationTimestamp: '2018-06-27T23:50:54Z',
          name: 'facefeed-proxy',
          namespace: '66e655a5-1564-48c4-8ab1-84f05b70615b',
          resourceVersion: '39027',
          selfLink:
            '/api/v1/namespaces/66e655a5-1564-48c4-8ab1-84f05b70615b/pods/facefeed-proxy',
          uid: 'ee4178c7-7a64-11e8-9219-506b8db6fe5b',
        },
        spec: {
          containers: [
            {
              args: ['tcp', '8888', 'facefeed-svc'],
              image:
                '770301640873.dkr.ecr.us-west-2.amazonaws.com/gcr_facefeed:latest',
              imagePullPolicy: 'Always',
              name: 'facefeed-tcp',
              ports: [
                {
                  containerPort: 8888,
                  hostPort: 8888,
                  name: 'tcp',
                  protocol: 'TCP',
                },
              ],
              resources: {},
              terminationMessagePath: '/dev/termination-log',
              terminationMessagePolicy: 'File',
              volumeMounts: [
                {
                  mountPath: '/var/run/secrets/kubernetes.io/serviceaccount',
                  name: 'default-token-w6spd',
                  readOnly: true,
                },
              ],
            },
          ],
          dnsPolicy: 'ClusterFirst',
          imagePullSecrets: [
            {
              name: 'face-feed',
            },
            {
              name: 'flask-web-hub',
            },
            {
              name: 'flaskweb',
            },
          ],
          nodeName: 'nos-nt-1',
          restartPolicy: 'Always',
          schedulerName: 'default-scheduler',
          securityContext: {},
          serviceAccount: 'default',
          serviceAccountName: 'default',
          terminationGracePeriodSeconds: 30,
          tolerations: [
            {
              effect: 'NoExecute',
              key: 'node.alpha.kubernetes.io/notReady',
              operator: 'Exists',
              tolerationSeconds: 300,
            },
            {
              effect: 'NoExecute',
              key: 'node.alpha.kubernetes.io/unreachable',
              operator: 'Exists',
              tolerationSeconds: 300,
            },
          ],
          volumes: [
            {
              name: 'default-token-w6spd',
              secret: {
                defaultMode: 420,
                secretName: 'default-token-w6spd',
              },
            },
          ],
        },
        status: {
          conditions: [
            {
              lastProbeTime: null,
              lastTransitionTime: '2018-06-27T23:50:54Z',
              status: 'True',
              type: 'Initialized',
            },
            {
              lastProbeTime: null,
              lastTransitionTime: '2018-06-27T23:51:21Z',
              status: 'True',
              type: 'Ready',
            },
            {
              lastProbeTime: null,
              lastTransitionTime: '2018-06-27T23:50:54Z',
              status: 'True',
              type: 'PodScheduled',
            },
          ],
          containerStatuses: [
            {
              containerID:
                'docker://48b4ddd95fd5d27a68cc96de8a9cff0a5e6e21489af0755b53d1d45868a1f24d',
              image:
                '770301640873.dkr.ecr.us-west-2.amazonaws.com/gcr_facefeed:latest',
              imageID:
                'docker-pullable://770301640873.dkr.ecr.us-west-2.amazonaws.com/gcr_facefeed@sha256:11441c75e3830fe0cb645c85556fd38b85b7ecaa4810f8b04d5ecfeafbb8587a',
              lastState: {},
              name: 'facefeed-tcp',
              ready: true,
              restartCount: 0,
              state: {
                running: {
                  startedAt: '2018-06-27T23:51:21Z',
                },
              },
            },
          ],
          hostIP: '10.15.232.22',
          phase: 'Running',
          podIP: '10.32.0.21',
          qosClass: 'BestEffort',
          startTime: '2018-06-27T23:50:54Z',
        },
      },
    ],
  },
};

// We start with multitenant sharing a single index 'mgmt-v1'
// which is always accessed via an alias 'mgmt' so we can migrate to new version as needed.
// For each tenant, we will create and use '<tenantId>' alias,
// this make each tenant appear to be logically separated
// and allow us to move a tenant to its own index down the road if the tenant
// data grows too large
export const GLOBAL_INDEX_NAME_INTERNAL = 'mgmt-v1'; // * This should only be used by code inside db-scripts/ *

export const EDGE_NAMES_SUBSET = [
  'SFO',
  'LAX',
  'ATL',
  'HOU',
  'ORD',
  'JFK',
  'BOM',
  'CDG',
  'LHR',
  'FRA',
];

export const AIRPORT_LOCATION_TYPE_VALUES = [
  'Terminal',
  'Kiosk',
  'POS',
  'Parking Lot',
  'Lobby',
  'Arrival',
  'Departure',
];

export const VIDEO_RESOLUTION_VALUES = ['High', 'Medium', 'Low'];

const jsCode = `
module.exports = async function(context) {
    return {
        status: 200,
        body: "Hello, world!\\n"
    };
}
`;

const pythonCode = `
def main():
    return "Hello, world!\\n"
`;

const goCode = `
package main

import (
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	msg := "Hello, World!"
	w.Write([]byte(msg))
}
`;

const rubyCode = `
# frozen_string_literal: true
def handler
  "Hello, world!\\n"
end
`;

const isCokePythonCode = `
#!/usr/bin/python

"""
Python fission script that takes an image as the input and outputs
whether a given object is present in the image or not

"""

import base64
import io
import argparse
import sys
import tensorflow as tf
import numpy as np
import json
from PIL import Image
from PIL import ImageDraw
#from utils import label_map_util

#PATH_TO_CKPT = '/root/IsCoke/output_inference_graph_174086.pb/frozen_inference_graph.pb'
PATH_TO_CKPT = '/app/frozen_inference_graph.pb'
#PATH_TO_LABELS = '/root/IsCoke/coke_label_map.pbtxt'

content_types = {'jpg': 'image/jpeg',
                 'jpeg': 'image/jpeg',
                 'png': 'image/png'}
extensions = sorted(content_types.keys())

def is_image(image):
    if not image:
        raise TypeError()
    elif image.split('.')[-1].lower() not in extensions:
        raise TypeError()
    return 1

class ObjectDetector(object):

    def __init__(self):
        self.detection_graph = self._build_graph()
        self.sess = tf.Session(graph=self.detection_graph)

#        label_map = label_map_util.load_labelmap(PATH_TO_LABELS)
 #       categories = label_map_util.convert_label_map_to_categories(
  #          label_map, max_num_classes=90, use_display_name=True)
   #     self.category_index = label_map_util.create_category_index(categories)

    def _build_graph(self):
        detection_graph = tf.Graph()
        with detection_graph.as_default():
            od_graph_def = tf.GraphDef()
            with tf.gfile.GFile(PATH_TO_CKPT, 'rb') as fid:
                serialized_graph = fid.read()
                od_graph_def.ParseFromString(serialized_graph)
                tf.import_graph_def(od_graph_def, name='')

        return detection_graph

    def _load_image_into_numpy_array(self, image):
        (im_width, im_height) = image.size
        return np.array(image.getdata()).reshape(
            (im_height, im_width, 3)).astype(np.uint8)

    def detect(self, image):
        image_np = self._load_image_into_numpy_array(image)
        image_np_expanded = np.expand_dims(image_np, axis=0)

        graph = self.detection_graph
        image_tensor = graph.get_tensor_by_name('image_tensor:0')
        boxes = graph.get_tensor_by_name('detection_boxes:0')
        scores = graph.get_tensor_by_name('detection_scores:0')
        classes = graph.get_tensor_by_name('detection_classes:0')
        num_detections = graph.get_tensor_by_name('num_detections:0')

        (boxes, scores, classes, num_detections) = self.sess.run(
            [boxes, scores, classes, num_detections],
            feed_dict={image_tensor: image_np_expanded})

        boxes, scores, classes, num_detections = map(
            np.squeeze, [boxes, scores, classes, num_detections])

        return boxes, scores, classes.astype(int), num_detections

def draw_bounding_box_on_image(image, box, color='red', thickness=4):
    draw = ImageDraw.Draw(image)
    im_width, im_height = image.size
    ymin, xmin, ymax, xmax = box
    (left, right, top, bottom) = (xmin * im_width, xmax * im_width,
                                  ymin * im_height, ymax * im_height)
    draw.line([(left, top), (left, bottom), (right, bottom),
               (right, top), (left, top)], width=thickness, fill=color)

def encode_image(image):
    image_buffer = io.BytesIO()
    image.save(image_buffer, format='PNG')
    imgstr = 'data:image/png;base64,{}'.format(
        base64.b64encode(image_buffer.getvalue()))
    return imgstr

#def decode_image(file):
 #  to_be_decoded = open(file, 'rb')
  # data = to_be_decoded.readline()
   #output_file = file + "_decode.png"
   #with open(output_file, 'wb') as f:
    #   dt = base64.b64decode(data)
     #  f.write(base64.b64decode(dt))
   #return

def detect_objects(payload):
    d1_image = base64.b64decode(payload)
    #d2_image = base64.b64decode(d1_image)
    image = Image.open(io.BytesIO(d1_image))
    boxes, scores, classes, num_detections = client.detect(image)
    image.thumbnail((480, 480), Image.ANTIALIAS)
    new_images = {}
    for i in range(int(num_detections)):
        if scores[i] < 0.85:
            continue
        cls = classes[i]
        if cls not in new_images.keys():
            new_images[cls] = image.copy()
        draw_bounding_box_on_image(new_images[cls], boxes[i],
                                   thickness=int(scores[i] * 10) - 4)
        #print ('Score: %s' % str(scores[i] * 100))

    result = {}
    result['original'] = encode_image(image.copy())
    result['Coke'] = None
    filecount = 0
    for cls, new_image in new_images.items():
        #new_image.save(str(filecount) + '_out_picture.png')
        filecount += 1
        result['Coke'] = encode_image(new_image)
    if filecount > 0:
        return filecount, result['Coke'], scores[0]
    else:
        return filecount, result['original'], 0

client = ObjectDetector()


def main():
    # call the detect_objects on the image file
    payload = request.get_data()
    #payload = incoming['Payload']
    found, boundedImage, score  = detect_objects(payload)
    data = {"result": found, "score": score, "image": boundedImage}
    json_data = json.dumps(data)
    return json_data

if __name__ == '__main__':
    main()

`;

const tempPythonCode = `
import json

#Filter temperature greater than 80F
def filterTemperature(payload):
    data = json.loads(payload)
    return (data["unit"]=="F") & (data["temperature"] >=80)

'''
Example payload
   payload = '{
    "deviceId": "D13",
    "temperature": 85,
    "unit": "F" }'
'''
def main():
   payload = request.get_data()
   if filterTemperature(payload):
         return payload
   return ""

if __name__ == '__main__':
    main()
`;

const jsSampleCode = `
//
// Example custom app to move data from Sherlock
// to custom destination
//
(function (global, undefined) {
	"use strict";
	undefinedVariable = {};
	undefinedVariable.prop = 5;

	function initializeProperties(target, members) {
		var keys = Object.keys(members);
		var properties;
		var i, len;
		for (i = 0, len = keys.length; i < len; i++) {
			var key = keys[i];
			var enumerable = key.charCodeAt(0) !== /*_*/95;
			var member = members[key];
			if (member && typeof member === 'object') {
				if (member.value !== undefined || typeof member.get === 'function' || typeof member.set === 'function') {
					if (member.enumerable === undefined) {
						member.enumerable = enumerable;
					}
					properties = properties || {};
					properties[key] = member;
					continue;
				}
			}
			if (!enumerable) {
				properties = properties || {};
				properties[key] = { value: member, enumerable: enumerable, configurable: true, writable: true }
				continue;
			}
			target[key] = member;
		}
		if (properties) {
			Object.defineProperties(target, properties);
		}
	}

	(function (rootNamespace) {

		// Create the rootNamespace in the global namespace
		if (!global[rootNamespace]) {
			global[rootNamespace] = Object.create(Object.prototype);
		}

		// Cache the rootNamespace we just created in a local variable
		var _rootNamespace = global[rootNamespace];
		if (!_rootNamespace.Namespace) {
			_rootNamespace.Namespace = Object.create(Object.prototype);
		}

		function defineWithParent(parentNamespace, name, members) {
			/// <summary locid="1">
			/// Defines a new namespace with the specified name, under the specified parent namespace.
			/// </summary>
			/// <param name="parentNamespace" type="Object" locid="2">
			/// The parent namespace which will contain the new namespace.
			/// </param>
			/// <param name="name" type="String" locid="3">
			/// Name of the new namespace.
			/// </param>
			/// <param name="members" type="Object" locid="4">
			/// Members in the new namespace.
			/// </param>
			/// <returns locid="5">
			/// The newly defined namespace.
			/// </returns>
			var currentNamespace = parentNamespace,
				namespaceFragments = name.split(".");

			for (var i = 0, len = namespaceFragments.length; i < len; i++) {
				var namespaceName = namespaceFragments[i];
				if (!currentNamespace[namespaceName]) {
					Object.defineProperty(currentNamespace, namespaceName,
						{ value: {}, writable: false, enumerable: true, configurable: true }
					);
				}
				currentNamespace = currentNamespace[namespaceName];
			}

			if (members) {
				initializeProperties(currentNamespace, members);
			}

			return currentNamespace;
		}

		function define(name, members) {
			/// <summary locid="6">
			/// Defines a new namespace with the specified name.
			/// </summary>
			/// <param name="name" type="String" locid="7">
			/// Name of the namespace.  This could be a dot-separated nested name.
			/// </param>
			/// <param name="members" type="Object" locid="4">
			/// Members in the new namespace.
			/// </param>
			/// <returns locid="5">
			/// The newly defined namespace.
			/// </returns>
			return defineWithParent(global, name, members);
		}

		// Establish members of the "WinJS.Namespace" namespace
		Object.defineProperties(_rootNamespace.Namespace, {

			defineWithParent: { value: defineWithParent, writable: true, enumerable: true },

			define: { value: define, writable: true, enumerable: true }

		});

	})("WinJS");

	(function (WinJS) {

		function define(constructor, instanceMembers, staticMembers) {
			/// <summary locid="8">
			/// Defines a class using the given constructor and with the specified instance members.
			/// </summary>
			/// <param name="constructor" type="Function" locid="9">
			/// A constructor function that will be used to instantiate this class.
			/// </param>
			/// <param name="instanceMembers" type="Object" locid="10">
			/// The set of instance fields, properties and methods to be made available on the class.
			/// </param>
			/// <param name="staticMembers" type="Object" locid="11">
			/// The set of static fields, properties and methods to be made available on the class.
			/// </param>
			/// <returns type="Function" locid="12">
			/// The newly defined class.
			/// </returns>
			constructor = constructor || function () { };
			if (instanceMembers) {
				initializeProperties(constructor.prototype, instanceMembers);
			}
			if (staticMembers) {
				initializeProperties(constructor, staticMembers);
			}
			return constructor;
		}

		function derive(baseClass, constructor, instanceMembers, staticMembers) {
			/// <summary locid="13">
			/// Uses prototypal inheritance to create a sub-class based on the supplied baseClass parameter.
			/// </summary>
			/// <param name="baseClass" type="Function" locid="14">
			/// The class to inherit from.
			/// </param>
			/// <param name="constructor" type="Function" locid="9">
			/// A constructor function that will be used to instantiate this class.
			/// </param>
			/// <param name="instanceMembers" type="Object" locid="10">
			/// The set of instance fields, properties and methods to be made available on the class.
			/// </param>
			/// <param name="staticMembers" type="Object" locid="11">
			/// The set of static fields, properties and methods to be made available on the class.
			/// </param>
			/// <returns type="Function" locid="12">
			/// The newly defined class.
			/// </returns>
			if (baseClass) {
				constructor = constructor || function () { };
				var basePrototype = baseClass.prototype;
				constructor.prototype = Object.create(basePrototype);
				Object.defineProperty(constructor.prototype, "_super", { value: basePrototype });
				Object.defineProperty(constructor.prototype, "constructor", { value: constructor });
				if (instanceMembers) {
					initializeProperties(constructor.prototype, instanceMembers);
				}
				if (staticMembers) {
					initializeProperties(constructor, staticMembers);
				}
				return constructor;
			} else {
				return define(constructor, instanceMembers, staticMembers);
			}
		}

		function mix(constructor) {
			/// <summary locid="15">
			/// Defines a class using the given constructor and the union of the set of instance members
			/// specified by all the mixin objects.  The mixin parameter list can be of variable length.
			/// </summary>
			/// <param name="constructor" locid="9">
			/// A constructor function that will be used to instantiate this class.
			/// </param>
			/// <returns locid="12">
			/// The newly defined class.
			/// </returns>
			constructor = constructor || function () { };
			var i, len;
			for (i = 0, len = arguments.length; i < len; i++) {
				initializeProperties(constructor.prototype, arguments[i]);
			}
			return constructor;
		}

		// Establish members of "WinJS.Class" namespace
		WinJS.Namespace.define("WinJS.Class", {
			define: define,
			derive: derive,
			mix: mix
		});

	})(WinJS);

})(this);
`;

const appPythonCode = `
#
# Simple app example script on Sherlock platform
#

import glob
import os
from PIL import Image


def make_image_thumbnail(filename):
    # The thumbnail will be named "<original_filename>_thumbnail.jpg"
    base_filename, file_extension = os.path.splitext(filename)
    thumbnail_filename = f"{base_filename}_thumbnail{file_extension}"

    # Create and save thumbnail image
    image = Image.open(filename)
    image.thumbnail(size=(128, 128))
    image.save(thumbnail_filename, "JPEG")

    return thumbnail_filename


# Loop through all jpeg files in the folder and make a thumbnail for each
for image_file in glob.glob("*.jpg"):
    thumbnail_file = make_image_thumbnail(image_file)

    print(f"A thumbnail for {image_file} was saved as {thumbnail_file}")
`;

export const MOCK_SCRIPTS = [
  {
    name: 'Temperature',
    type: 'Transformation',
    language: 'python',
    environment: 'python-env',
    code: tempPythonCode,
    params: [],
  },
  {
    name: 'Object Recognition',
    type: 'Transformation',
    language: 'python',
    environment: 'tensorflow-python',
    code: isCokePythonCode,
    params: [],
  },
  {
    name: 'Image Processing',
    type: 'Transformation',
    language: 'python',
    environment: 'tensorflow-python',
    code: isCokePythonCode,
    params: [],
  },
  {
    name: 'Data Extraction',
    type: 'Transformation',
    language: 'python',
    environment: 'python-env',
    code: isCokePythonCode,
    params: [],
  },
  // {
  //   name: 'Custom Data Mover',
  //   type: 'Function',
  //   language: 'javascript',
  //   environment: 'node-env',
  //   code: jsSampleCode,
  // },
  {
    name: 'Simple App',
    type: 'Function',
    language: 'python',
    environment: 'python-env',
    code: appPythonCode,
    params: [],
  },
  // {
  //   name: 'transform-08',
  //   type: 'Transformation',
  //   language: 'javascript',
  //   environment: 'node-env',
  //   code: jsCode,
  // },
  // {
  //   name: 'lambda-13',
  //   type: 'Function',
  //   language: 'javascript',
  //   environment: 'node-env',
  //   code: jsCode,
  // },
  // {
  //   name: 'lambda-78',
  //   type: 'Function',
  //   language: 'go',
  //   environment: 'go-env',
  //   code: goCode,
  // },
  // {
  //   name: 'lambda-18',
  //   type: 'Function',
  //   language: 'ruby',
  //   environment: 'ruby-env',
  //   code: rubyCode,
  // },
  // {
  //   name: 'transform-02',
  //   type: 'Transformation',
  //   language: 'javascript',
  //   environment: 'node-env',
  //   code: jsCode,
  // },
  // {
  //   name: 'lambda-04',
  //   type: 'Function',
  //   language: 'python',
  //   environment: 'python-env',
  //   code: pythonCode,
  // },
  // {
  //   name: 'transform-17',
  //   type: 'Transformation',
  //   language: 'javascript',
  //   environment: 'node-env',
  //   code: jsCode,
  // },
  // {
  //   name: 'lambda-83',
  //   type: 'Function',
  //   language: 'javascript',
  //   environment: 'node-env',
  //   code: jsCode,
  // },
];

export const MOCK_DATA_SOURCES = [
  {
    name: 'sensor-001',
    type: 'Sensor',
    sensorModel: 'Model 3',
    connection: 'Secure',
    fields: [],
    selectors: [],
  },
  {
    name: 'sensor-002',
    type: 'Sensor',
    sensorModel: 'Model X',
    connection: 'Secure',
    fields: [],
    selectors: [],
  },
  {
    name: 'SJC-1-b-gateway',
    type: 'Gateway',
    sensorModel: 'Model S',
    connection: 'Unsecure',
    fields: [],
    selectors: [],
  },
  {
    name: 'sensor-004',
    type: 'Sensor',
    sensorModel: 'Model S',
    connection: 'Secure',
    fields: [],
    selectors: [],
  },
  {
    name: 'sensor-011',
    type: 'Sensor',
    sensorModel: 'Model 3',
    connection: 'Unsecure',
    fields: [],
    selectors: [],
  },
  {
    name: 'gateway-03',
    type: 'Gateway',
    sensorModel: 'Model 3',
    connection: 'Secure',
    fields: [],
    selectors: [],
  },
  {
    name: 'sensor-053',
    type: 'Sensor',
    sensorModel: 'Model 3',
    connection: 'Secure',
    fields: [],
    selectors: [],
  },
  {
    name: 'sensor-044',
    type: 'Sensor',
    sensorModel: 'Model S',
    connection: 'Secure',
    fields: [],
    selectors: [],
  },
  {
    name: 'sensor-099',
    type: 'Sensor',
    sensorModel: 'Model X',
    connection: 'Unsecure',
    fields: [],
    selectors: [],
  },
  {
    name: 'gateway-02',
    type: 'Gateway',
    sensorModel: 'Model X',
    connection: 'Unsecure',
    fields: [],
    selectors: [],
  },
  {
    name: 'sensor-075',
    type: 'Sensor',
    sensorModel: 'Model 3',
    connection: 'Secure',
    fields: [],
    selectors: [],
  },
];

// TODO: if DataStream origin is another DataStream, we need to allow specifying origin DataStream name / id.
// TODO: one way to do this is to have special <entity type / id> category type, whose value can be any id
export const MOCK_DATA_STREAMS: any[] = [
  {
    name: 'security-camera-video',
    dataType: 'Image',
    origin: 'Data Source',
    originSelectors: [],
    destination: DataStreamDestination.Edge,
    edgeStreamType: EdgeStreamType.Kafka,
    size: 0,
    enableSampling: false,
    transformationArgsList: [],
    dataRetention: [],
  },
  {
    name: 'person-of-interest-edge',
    dataType: 'Image',
    origin: 'Data Source', // POI-detectors (5000 devices)
    originSelectors: [],
    destination: DataStreamDestination.Edge,
    edgeStreamType: EdgeStreamType.Kafka,
    size: 0,
    enableSampling: true,
    samplingInterval: 15,
    transformationArgsList: [],
    dataRetention: [], // Up to 200.0 TB
  },
  {
    name: 'passenger-checking-local',
    dataType: 'Image',
    origin: 'Data Source', // gate-kiosks (2500 devices)
    originSelectors: [],
    destination: DataStreamDestination.Edge,
    edgeStreamType: EdgeStreamType.ElasticSearch,
    size: 0,
    enableSampling: false,
    samplingInterval: 0, // All
    transformationArgsList: [],
    dataRetention: [], // 8 Months
  },
  {
    name: 'person-of-interest-cloud',
    dataType: 'Image',
    origin: 'Data Stream', // POI-edge
    originSelectors: [],
    originIndex: 1,
    destination: DataStreamDestination.Cloud, // Google Cloud Storage
    cloudType: CloudType.AWS,
    awsStreamType: AWSStreamType.DynamoDB,
    cloudCredsId: null,
    awsCloudRegion: AWS_REGION.US_WEST_2,
    gcpCloudRegion: null,
    size: 0,
    enableSampling: true,
    samplingInterval: 60,
    transformationArgsList: [],
    dataRetention: [], // Up to 60.0 TB
  },
  {
    name: 'passenger-traffic-training',
    dataType: 'Processed',
    origin: 'Data Stream', // passenger-checkin-local
    originSelectors: [],
    originIndex: 2,
    destination: DataStreamDestination.Cloud, // Google Cloud Storage
    cloudType: CloudType.GCP,
    gcpStreamType: GCPStreamType.PubSub,
    cloudCredsId: null,
    awsCloudRegion: null,
    gcpCloudRegion: GCP_REGION.US_WEST1,
    size: 0,
    enableSampling: false,
    samplingInterval: 0, // Custom
    transformationArgsList: [],
    dataRetention: [], // Up to 20.0 TB
  },
];
