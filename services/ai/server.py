
from concurrent import futures
import time
import math
import logging

import grpc

from proto import mlmodel_pb2
from proto import mlmodel_pb2_grpc
import urllib.request
import zipfile
import random
import os
import tensorflow as tf
import sys
import shutil
from multiprocessing import Pool



class MLModelServicer(mlmodel_pb2_grpc.MLModelServicer):
    """Provides methods that implement functionality of ml model server."""

    def __init__(self):
        self.folder_base = "/tmp"
    '''
    Validate method will download the zip file from the url and extracts it to a folder.
    It then calls the validate model method in a separate process.If the ML model is not a valid file,
    then it returns an error message. 
    Note: Tensorflow graph session is not releasing memory,so running it as a separate process.
    '''
    def Validate(self, request, context):
        model_url = request.url
        mlmodel_type = request.mlmodel_type
        logging.info("Received ml model validate request for url " + model_url)
        valid_model = True
        error_msg = ""
        p = None
        try:
            file_path = self.folder_base+"/"+str(random.randint(1, 10000))
            if not os.path.exists(file_path):
                os.makedirs(file_path)
            file_name = file_path+"/temp.zip"
            urllib.request.urlretrieve(model_url, file_name)
            if os.path.getsize(file_name) != request.mlmodel_size_bytes:
                msg = "Downloaded model file size {} doesnot match the input mlmodel_size_bytes {}".format(os.path.getsize(file_name), request.mlmodel_size_bytes
                                                                                                           )
                logging.error(msg)
                raise Exception(msg)
            with zipfile.ZipFile(file_name, "r") as zip_ref:
                zip_ref.extractall(file_path)
            logging.info("Successfully extracted model zip file. Url: " + model_url)
            p = Pool(processes=1)
            if mlmodel_type == mlmodel_pb2.TENSORFLOW_1_13_1:
                result = p.apply(self.validate_tensorflow_model,args=(request, file_path))
                valid_model = result[0]
                error_msg =  result[1]
            elif mlmodel_type == mlmodel_pb2.OPENVINO_2019_R2:
                (valid_model,error_msg) = self.validate_openvino_model(request, file_path)
            logging.info("Done validating ml model. Url: " + model_url)
        except Exception as e:
            valid_model = False
            logging.error(e)
            error_msg = str(e)
        finally:
            if os.path.exists(file_path):
                shutil.rmtree(file_path)
            if p is not None:
                p.close()
        return mlmodel_pb2.ValidateResponse(valid_model=valid_model, error_msg=error_msg)

    #Validate tensorflow model by loading the graph in the tf session.
    def validate_tensorflow_model(self,request,model_file_path):
        try:
            with tf.Session(graph=tf.Graph()) as sess:
                tf.saved_model.loader.load(sess, ["serve"], model_file_path)
        except Exception as e:
            logging.error(e)
            error_msg = str(e)
            return (False,error_msg)
        return (True,"")

    #Currently only validates the file extensions.
    #TODO: Openvino has no way to validate the model without running it on movidus.
    # We should figure out different approach here. 
    def validate_openvino_model(self,request,model_file_path):
        bin_file_exists = False
        xml_file_exists = False
        for fname in os.listdir(model_file_path):
            if fname.endswith('.bin'):
                bin_file_exists = True
            if fname.endswith('.xml'):
                xml_file_exists = True
        if bin_file_exists == False or xml_file_exists == False:
            error_msg = "Zip file doesn't contain either .bin or .xml file"
            logging.error(error_msg)
            return (False,error_msg)
        return (True,"")



def serve():
    #max threads is set to 1 ,so all api calls are executed sequentially.
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=1))
    mlmodel_pb2_grpc
    mlmodel_pb2_grpc.add_MLModelServicer_to_server(
        MLModelServicer(), server)
    server.add_insecure_port('[::]:8500')
    server.start()
    logging.info("Started server")
    try:
        while True:
            # One day - 60*60*24
            time.sleep(86400)
    except KeyboardInterrupt:
        server.stop(0)


if __name__ == '__main__':
    logging.basicConfig(stream=sys.stdout, level=logging.INFO)
    serve()
