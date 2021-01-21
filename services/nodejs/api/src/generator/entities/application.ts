import * as uuidv4 from 'uuid/v4';
import { randomAttribute, pickMany } from '../common';

export function randomApplication(ctx: any, apiVersion: string, project: any) {
  const { tenantId } = ctx;
  const id = uuidv4();
  const yamlId = uuidv4();
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const yamlData = `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: deployment-demo-${yamlId}
spec:
  selector:
    matchLabels:
      demo: deployment
  replicas: 5
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
    type: RollingUpdate
  template:
    metadata:
      labels:
        demo: deployment
        version: v1
    spec:
      containers:
      - name: busybox
        image: busybox
        command: [ "sh", "-c", "while true; do echo hostname; sleep 60; done" ]
        volumeMounts:
        - name: content
          mountPath: /data
      - name: nginx
        image: nginx
        volumeMounts:
          - name: content
            mountPath: /usr/share/nginx/html
            readOnly: true
      volumes:
      - name: content`;
  const projectId = project.id;
  const edgeIds = pickMany(project.edgeIds).sort();

  if (apiVersion === 'v1') {
    return {
      id,
      tenantId,
      name,
      description,
      yamlData,
      projectId,
      edgeIds,
    };
  } else {
    return {
      id,
      tenantId,
      name,
      description,
      projectId,
      edgeIds,
      appManifest: yamlData,
    };
  }
}

export function randomApplicationUpdate(
  ctx: any,
  apiVersion: string,
  project: any,
  entity
) {
  const updated = randomApplication(ctx, apiVersion, project);
  const { id } = entity;
  return { ...updated, id };
}

export function purifyApplication(application: any, apiVersion: string) {
  if (apiVersion === 'v1') {
    const {
      id,
      tenantId,
      name,
      description,
      yamlData,
      projectId,
      edgeIds,
    } = application;
    edgeIds.sort();
    return {
      id,
      tenantId,
      name,
      description,
      yamlData,
      projectId,
      edgeIds,
    };
  } else {
    const {
      id,
      tenantId,
      name,
      description,
      appManifest,
      projectId,
      edgeIds,
    } = application;
    edgeIds.sort();
    return {
      id,
      tenantId,
      name,
      description,
      appManifest,
      projectId,
      edgeIds,
    };
  }
}
