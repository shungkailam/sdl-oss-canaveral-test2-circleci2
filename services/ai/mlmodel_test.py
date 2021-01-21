
from __future__ import print_function

import random
import logging

import grpc

from proto import mlmodel_pb2
from proto import mlmodel_pb2_grpc

'''
Example 
url = https://s3-us-west-2.amazonaws.com/sherlock-object-detection-model/saved_model.zip
size = 62623947

url=http://www.mynikko.com/dummy/dummy11.zip
size=117614
'''


def validate_tf_ml_model_pos_test(stub):
    validate_req = mlmodel_pb2.ValidateRequest(
        url="https://s3-us-west-2.amazonaws.com/sherlock-object-detection-model/saved_model.zip",
        mlmodel_size_bytes=62623947,
        mlmodel_type=mlmodel_pb2.TENSORFLOW_1_13_1)
    validate_resp = stub.Validate(validate_req)
    assert validate_resp.valid_model == True
    print(validate_resp)


def validate_tf_ml_model_neg_test(stub):
    validate_req = mlmodel_pb2.ValidateRequest(
        url="http://www.mynikko.com/dummy/dummy11.zip",
        mlmodel_size_bytes=117614,
        mlmodel_type=mlmodel_pb2.TENSORFLOW_1_13_1)
    validate_resp = stub.Validate(validate_req)
    assert validate_resp.valid_model == False
    print(validate_resp.error_msg)

def validate_openvino_ml_model_pos_test(stub):
    validate_req = mlmodel_pb2.ValidateRequest(
        url="https://sherlock-openvino.s3-us-west-2.amazonaws.com/face-detection-R2.zip",
        mlmodel_size_bytes=1959673,
        mlmodel_type=mlmodel_pb2.OPENVINO_2019_R2)
    validate_resp = stub.Validate(validate_req)
    assert validate_resp.valid_model == True
    print(validate_resp)


def validate_openvino_ml_model_neg_test(stub):
    validate_req = mlmodel_pb2.ValidateRequest(
        url="http://www.mynikko.com/dummy/dummy11.zip",
        mlmodel_size_bytes=117614,
        mlmodel_type=mlmodel_pb2.OPENVINO_2019_R2)
    validate_resp = stub.Validate(validate_req)
    assert validate_resp.valid_model == False
    print(validate_resp.error_msg)


def run():
    with grpc.insecure_channel('localhost:8500') as channel:
        stub = mlmodel_pb2_grpc.MLModelStub(channel)
        validate_tf_ml_model_pos_test(stub)
        validate_tf_ml_model_neg_test(stub)
        validate_openvino_ml_model_pos_test(stub)
        validate_openvino_ml_model_neg_test(stub)


if __name__ == '__main__':
    logging.basicConfig()
    run()
