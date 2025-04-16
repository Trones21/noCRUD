"""Defines the routes for our API."""

from rest_framework.routers import SimpleRouter
from django.shortcuts import render
from django.urls import path
from django.views.decorators.http import require_http_methods

from api.models.users import UsersViewSet
from api.models.user_role import UserRoleViewSet
from api.models.roles import RolesViewSet

from .views import login_view, get_model_shape
from example_app.storage import get_file_view
from example_app.permissions import (
    get_user_permissions,
    get_all_permissions,
    get_permissions,
)


@require_http_methods(["GET"])
def index(request):
    """Returns the correct endpoint according to the routes."""
    return render(request, "index.html")


router = SimpleRouter()

router.register(r"users", UsersViewSet)
router.register(r"users_roles", UserRoleViewSet)
router.register(r"roles", RolesViewSet)


urlpatterns = [
    path("", index, name="index"),
    path("login/", login_view, name="login"),
    path("get_file/", get_file_view, name="get-file"),
    path("get_user_permissions/", get_user_permissions, name="get-user-permissions"),
    path("get_permissions/", get_permissions, name="get-permissions"),
    path("get-all-permissions/", get_all_permissions, name="get-all-permissions"),
    path("model/<str:model_name>/", get_model_shape, name="model-shape"),
]

urlpatterns += router.urls
