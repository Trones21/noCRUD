import json
import io
import os
import binascii
import mimetypes
from types import SimpleNamespace
from datetime import datetime
from django.http import HttpRequest, JsonResponse, Http404
from minio import Minio
from example_app import settings
from django.views.decorators.http import require_POST

client = Minio(
    endpoint=settings.OBJECT_STORAGE["endpoint"],
    access_key=settings.OBJECT_STORAGE["access_key"],
    secret_key=settings.OBJECT_STORAGE["secret_key"],
    secure=settings.OBJECT_STORAGE["secure"],
)


@require_POST
def get_file(request: HttpRequest) -> JsonResponse:
    if settings.STORAGE_MODE == settings.StorageMode.LOCAL:
        return get_file_local(request)
    if settings.STORAGE_MODE == settings.StorageMode.S3:
        return get_file_S3(request)
    raise ValueError(f"Unsupported STORAGE_MODE: {settings.STORAGE_MODE}")


get_file_view = require_POST(get_file)


def upload_file(file, folder_name):
    if settings.STORAGE_MODE == settings.StorageMode.LOCAL:
        return save_file_local(file, folder_name)
    if settings.STORAGE_MODE == settings.StorageMode.S3:
        return upload_file_S3(file, folder_name)


def get_file_local(request: HttpRequest) -> JsonResponse:
    data = json.loads(request.body)
    file_path = (
        data["bucket"] + "/" + data["object_name"]
        if data["bucket"]
        else data["object_name"]
    )
    full_file_path = os.path.join(settings.BASE_DIR, "local_file_storage", file_path)
    if not os.path.exists(full_file_path):
        raise Http404("File not found")

    try:
        content_type, _ = mimetypes.guess_type(full_file_path)
        if not content_type:
            content_type = "application/octet-stream"

        with open(full_file_path, "rb") as file:
            file_content = file.read()
        file_hex = binascii.hexlify(file_content).decode("utf-8")

        response_data = {"file": file_hex, "content_type": content_type}
        return JsonResponse(response_data)
    except Exception as e:
        print("err", e)
        raise Http404(f"Error serving file: {str(e)}")


def save_file_local(file, dirPath):
    # Check for base directory
    local_storage_base = os.path.join(settings.BASE_DIR, "local_file_storage")
    if not os.path.exists(local_storage_base):
        os.makedirs(local_storage_base)

    # Check for full directory
    full_dir_path = os.path.join(local_storage_base, dirPath.strip("/"))
    if not os.path.exists(full_dir_path):
        os.makedirs(full_dir_path)

    # Construct the full file path
    file_path = os.path.join(full_dir_path, file.name)

    # Save the file
    with open(file_path, "wb+") as destination:
        for chunk in file.chunks():
            destination.write(chunk)

    result = SimpleNamespace(file_path=file_path, version_id=datetime.now().isoformat())
    return result


@require_POST
def get_file_S3(request: HttpRequest) -> JsonResponse:
    bucket = settings.OBJECT_STORAGE["top_level_bucket"]
    data = json.loads(request.body)
    objectUri = (
        data["bucket"] + "/" + data["object_name"]
        if data["bucket"]
        else data["object_name"]
    )
    bucketObject = client.get_object(bucket, objectUri, version_id=data["version_id"])
    return JsonResponse(
        {
            "file": bucketObject.data.hex(),
            "content_type": bucketObject.headers["Content-Type"],
        }
    )


def upload_file_S3(file, folder_name):
    bucket = settings.OBJECT_STORAGE["top_level_bucket"]
    if not client.bucket_exists(bucket):
        print(
            "Bucket ( "
            + bucket
            + " ) does not exist. Check your OBJECT_STORAGE object in settings.py "
        )
        return False
    objectUri = f"{folder_name}/{file._name}" if folder_name else file.name
    fileData = file.read()
    result = client.put_object(
        bucket,
        objectUri,
        io.BytesIO(fileData),
        length=io.BytesIO(fileData).getbuffer().nbytes,
        content_type=file.content_type,
    )
    return result
