import json

from django.apps import apps
from django.contrib.auth import authenticate, login
from django.http import JsonResponse
from django.views.decorators.csrf import csrf_exempt
from rest_framework.decorators import api_view
from api.models.users import Users


@csrf_exempt
def login_view(request):
    data = json.loads(request.body)

    username = data.get("username")
    password = data.get("password")
    if username is None or password is None:
        return JsonResponse(
            {"detail": "Please provide username and password."}, status=400
        )

    user = authenticate(username=username, password=password)

    if user is None:
        return JsonResponse({"detail": "Invalid credentials."}, status=400)

    assert isinstance(user, Users)

    jsonResponse = {
        "user": {
            "id": user.id,
            "username": user.username,
            "first_name": user.first_name,
            "last_name": user.last_name,
        },
        "detail": "Successfully logged in.",
    }

    login(request, user)

    return JsonResponse(jsonResponse)


@api_view(["GET"])
def get_model_shape(request, model_name):
    try:
        model = apps.get_model(app_label="api", model_name=model_name)
        if model:
            model_fields = model._meta.get_fields()
            fields_info = [
                {"name": field.name, "type": field.get_internal_type()}
                for field in model_fields
            ]
            return JsonResponse({"fields": fields_info})
    except LookupError:
        pass

    models = [model.__name__ for model in apps.get_models()]
    return JsonResponse({"error": "Model not found", "models": models}, status=404)
