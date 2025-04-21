"""This module extends the base Django user module with customizations for our use."""

from django.db import models
from django.conf import settings
from django.contrib.auth.models import AbstractUser
from rest_framework.viewsets import ModelViewSet
from rest_framework import serializers
from rest_framework.request import Request as DRFRequest
from rest_framework.serializers import ModelSerializer


class Users(AbstractUser):
    """The model for a user.
    AbstractUser provides built in fields:
    id,
    password,
    is_superuser,
    last_login,
    is_active,
    date_joined,
    email,
    user_permissions
    """

    id: int
    REQUIRED_FIELDS = ["first_name", "last_name"]
    USERNAME_FIELD = "username"

    username = models.CharField(null=False, unique=True, max_length=50)
    phone = models.TextField(blank=True, null=True)
    roles = models.ManyToManyField("api.Roles", through="api.UserRole")
    preferences = models.JSONField(blank=True, null=True)

    class Meta:
        """Model settings."""

        managed = True
        db_table = "users"


class UsersSerializer(ModelSerializer):
    """The serializer for the User model."""

    class Meta:
        """Model settings."""

        model = Users
        fields = "__all__"

    if settings.DEPLOYMENT_TYPE != "DEVELOPMENT":
        password = serializers.CharField(write_only=True)

    def create(self, raw_data):
        """Validation of raw data occurs on create, enforced by AbstractUser model"""
        user = Users.objects.create(**raw_data)
        user.set_password(raw_data["password"])
        user.save()
        return user


class UsersViewSet(ModelViewSet):
    """The UserViewSet model."""

    request: DRFRequest
    queryset = Users.objects.order_by("pk")
    serializer_class = UsersSerializer

    def get_queryset(self):
        queryset = Users.objects.order_by("pk")
        username = self.request.query_params.get("username")
        if username:
            queryset = queryset.filter(username=username)
        return queryset

    def update(self, request, pk=None):
        # Note: Roles and permissions are updated via their own endpoints
        return super().update(request, pk)
