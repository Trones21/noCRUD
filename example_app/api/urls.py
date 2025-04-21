"""Defines the routes for our API."""

from rest_framework.routers import SimpleRouter
from django.shortcuts import render
from django.urls import path
from django.views.decorators.http import require_http_methods

from api.models.users import UsersViewSet
from api.models.user_role import UserRoleViewSet
from api.models.roles import RolesViewSet
from api.models.actor import ActorViewSet
from api.models.character import CharacterViewSet
from api.models.production import ProductionViewSet
from api.models.tags_and_tagcategories import TagCategoryViewSet, TagViewSet
from api.models.universe import UniverseViewSet
from api.models.vote import VoteViewSet
from api.models.backstory_pitch import BackstoryPitchViewSet

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
router.register(r"pitch", BackstoryPitchViewSet)
router.register(r"universe", UniverseViewSet)
router.register(r"production", ProductionViewSet)
router.register(r"actor", ActorViewSet)
router.register(r"character", CharacterViewSet)
router.register(r"vote", VoteViewSet)
router.register(r"tagcategory", TagCategoryViewSet)
router.register(r"tag", TagViewSet)

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
