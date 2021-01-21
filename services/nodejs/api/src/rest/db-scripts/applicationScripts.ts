export const faceRecogitionScript = `import base64
import cStringIO
import json
import io
from PIL import Image
import tensorflow as tf
import numpy as np
from facerecognition import facenet
from facerecognition.align import detect_face
import cv2
import logging


# some constants kept as default from facenet
minsize = 20
threshold = [0.6, 0.7, 0.7]
factor = 0.709
margin = 44
input_image_size = 160


sess = tf.Session()

# read pnet, rnet, onet models from align directory and files are det1.npy, det2.npy, det3.npy
pnet, rnet, onet = detect_face.create_mtcnn(sess, '/mllib/facerecognition/align')

# read 20180402-114759 model file downloaded from https://drive.google.com/file/d/1EXPBSXwTaqrSC0OhUdXNmKSh9qJUQ55-/view
facenet.load_model("/mllib/facerecognition/20180402-114759/20180402-114759.pb")

# Get input and output tensors
images_placeholder = tf.get_default_graph().get_tensor_by_name("input:0")
embeddings = tf.get_default_graph().get_tensor_by_name("embeddings:0")
phase_train_placeholder = tf.get_default_graph().get_tensor_by_name("phase_train:0")
embedding_size = embeddings.get_shape()[1]

def getFaces(img):
    faces = []
    img_size = np.asarray(img.shape)[0:2]
    bounding_boxes, _ = detect_face.detect_face(img, minsize, pnet, rnet, onet, threshold, factor)
    if not len(bounding_boxes) == 0:
        for face in bounding_boxes:
            if face[4] > 0.50:
                det = np.squeeze(face[0:4])
                bb = np.zeros(4, dtype=np.int32)
                bb[0] = np.maximum(det[0] - margin / 2, 0)
                bb[1] = np.maximum(det[1] - margin / 2, 0)
                bb[2] = np.minimum(det[2] + margin / 2, img_size[1])
                bb[3] = np.minimum(det[3] + margin / 2, img_size[0])
                cropped = img[bb[1]:bb[3], bb[0]:bb[2], :]
                resized = cv2.resize(cropped, (input_image_size,input_image_size),interpolation=cv2.INTER_CUBIC)
                prewhitened = facenet.prewhiten(resized)
                embedding = getEmbedding(prewhitened)
                listEmbedding = embedding.tolist()
                # encode image as jpeg
                _, img_encoded = cv2.imencode('.jpg', resized)
                encodedStr = base64.b64encode(img_encoded)
                top = np.int32(bb[0]).item()
                left = np.int32(bb[1]).item()
                bottom = np.int32(bb[2]).item()
                right = np.int32(bb[3]).item()
                faces.append({'face':encodedStr,'bb':[top,left,bottom,right],'embedding':listEmbedding})
    return faces
def getEmbedding(resized):
    reshaped = resized.reshape(-1,input_image_size,input_image_size,3)
    feed_dict = {images_placeholder: reshaped, phase_train_placeholder: False}
    embedding = sess.run(embeddings, feed_dict=feed_dict)
    return embedding

def detect_faces(data):
  #image = Image.open(image_path).convert('RGB')
  image = Image.open(io.BytesIO(base64.b64decode(data))).convert('RGB')
  cvImage = cv2.cvtColor(np.array(image), cv2.COLOR_RGB2BGR)
  return getFaces(cvImage)


def main(ctx,msg):
    logging.info("***** Face Recognition script Start *****")
    input_data = json.loads(msg)
    faces = detect_faces(input_data['data'])
    input_data['faces'] = faces
    logging.info("Detected number of faces: %d",len(faces))
    logging.info("***** Face Recognition script End *****")
    return ctx.send(json.dumps(input_data))

'''
#Test
if __name__ == '__main__':
    with open("/mllib/facerecognition/test.jpg", "rb") as image_file:
      encoded_string = base64.b64encode(image_file.read())
    faces = detect_faces(encoded_string)
    print type(faces[0]['embedding'])
    jsonStr = json.dumps(faces)
    temp = json.loads(jsonStr)
    image = Image.open(io.BytesIO(base64.b64decode(temp[0]['face']))).convert('RGB')
    cvImage = cv2.cvtColor(np.array(image), cv2.COLOR_RGB2BGR)
    cv2.imwrite('./result.jpg', cvImage)
'''`;

export const faceMatchScript = `import numpy as np
from PIL import Image
import logging
import cv2
import base64
import io
import time
import json
from threading import Thread
from elasticsearch import Elasticsearch
from elasticsearch_dsl import Search
import sys
from facerecognition import facenet


#esIP = "10.15.232.225"
esIP = "elasticsearch.default.svc.cluster.local"
esPort = 9200
esIndex = "datastream-faceregister"
threshold = 0.4

class FaceMatch(object):

    def __init__(self,threshold):
        self.knownfaces = []
        self.threshold = threshold

    def update_known_faces(self,faces):
        self.knownfaces = faces

    def match(self,face):
        for known_face in self.knownfaces:
            #dist = np.sqrt(np.sum(np.square(np.subtract(known_face['embedding'], face))))
            dist = facenet.distance(known_face['embedding'],face,1)
            logging.info("Calculated distance (%f) with employee id:%s",dist,known_face['employee_id'])
            if dist <= self.threshold:
                logging.info("Found matching face with distance: %f",dist)
                return known_face
        return 

class FetchKnownFaces(Thread):
    def __init__(self,esIP,esPort,esIndex,facematch):
        Thread.__init__(self)
        self.esIndex = esIndex
        self.esIP = esIP
        self.esPort = esPort
        self.facematch = facematch
    def connect(self):
        self.esclient = Elasticsearch([{'host': self.esIP, 'port': self.esPort}])

    def run(self):
        s = Search(using=self.esclient, index=self.esIndex)
        count =0
        while True:
            try:
                response = s.execute(True)
                if count % 10 == 0:
                    count = 0
                    logging.info("Fetched registered faces from Elastic Search. Number of records found: %d",len(response))
                facematch.update_known_faces(response)
                count = count +1
            except Exception as e:
                logging.exception("Failed to get registered faces from Elastic Search.")
            # Sleep for 60 secs
            time.sleep(60)

facematch = FaceMatch(threshold)
updateThread = FetchKnownFaces(esIP,esPort,esIndex,facematch)
updateThread.setDaemon(True)
updateThread.connect()
updateThread.start()

def main(ctx,msg):
    logging.info("***** Face Match script Start *****")
    msg = json.loads(msg)
    data = msg['data']
    image = Image.open(io.BytesIO(base64.b64decode(data))).convert('RGB')
    cvImage = cv2.cvtColor(np.array(image), cv2.COLOR_RGB2BGR)
    faces = msg['faces']
    # Returning if we don't find any face.  
    if len(faces) == 0:
        logging.info("No face found")
        logging.info("***** Face match script End *****")
        return
    for i in range(len(faces)):
        known_face = facematch.match(np.asarray(faces[i]['embedding']))
        bb = faces[i]['bb']
        if known_face is None:
            faces[i]['knownface'] = False
            cv2.rectangle(cvImage,(bb[0], bb[1]), (bb[2], bb[3]),(0, 0,255), 2)
        else:
            faces[i]['knownface'] = True
            faces[i]['name'] = known_face['name']
            faces[i]['designation'] = known_face['designation']
            faces[i]['department'] = known_face['department']
            faces[i]['employee_id'] = known_face['employee_id']
            logging.info("Found matching face with employee id: %s",known_face['employee_id'])
            cv2.rectangle(cvImage,(bb[0], bb[1]), (bb[2], bb[3]),(0, 255, 0), 2)
    response ={}
    if len(faces) !=0 :
        # encode image as jpeg
        _, img_encoded = cv2.imencode('.jpg', cvImage)
        encodedStr = base64.b64encode(img_encoded)
        response['image'] = encodedStr
        response['faces'] = faces
        logging.info("Idenitfied %d faces",len(faces))
    else:
        response['image'] = data
        response['faces'] = faces
    ctx.send(json.dumps(response))
    logging.info("***** Face match script End *****")
    return


'''
#Test
if __name__ == '__main__':
    faces = json.load(open('../tests/test.json'))
    for i in range(len(data)):
        known_face = facematch.match(np.asarray(data[i]['embedding']))
        if known_face is None:
            data[i]['knownface'] = False
        else:
            data[i]['knownface'] = True
            data[i]['name'] = known_face[i]['name']
        logging.info(data[i]) 
'''`;

export const customDataMoverScript = `//
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

export const dataExtractionScript = `#!/usr/bin/python

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
from flask import request
from flask import current_app
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
export const imageProcessingScript = `
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
from flask import request
from flask import current_app
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

export const objectRecognitionScript = `#!/usr/bin/python

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
from flask import request
from flask import current_app
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

export const simpleAppScript = `#
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

export const temperatureScript = `import json
from flask import request
from flask import current_app

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
