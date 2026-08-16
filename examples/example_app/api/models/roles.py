"""The model, serializer, and viewset for roles."""

from django.db import models
from django.contrib.auth.decorators import permission_required, login_required
from django.contrib.auth.models import Permission
from django.utils.decorators import method_decorator

from rest_framework.serializers import ModelSerializer
from rest_framework.viewsets import ModelViewSet
from rest_framework.response import Response
from rest_framework import status


class Roles(models.Model):
    """The model."""

    role_id = models.AutoField(primary_key=True)
    name = models.TextField(blank=False, null=False)
    description = models.TextField(blank=True, null=True)
    permissions = models.ManyToManyField(Permission)

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "roles"


class RolesSerializer(ModelSerializer):
    """The serializer class."""

    def validate(self, data):
        return data

    def create(self, validated_data):
        return super().create(validated_data)

    class Meta:
        """Settings for the serializer."""

        model = Roles
        fields = "__all__"
        extra_kwargs = {
            "permissions": {"required": False, "allow_empty": True},
        }


class RolesViewSet(ModelViewSet):
    """The viewset."""

    queryset = Roles.objects.order_by("pk")
    serializer_class = RolesSerializer
    # authentication_classes = [JWTAccessTokenAuthentication]

    # @method_decorator(login_required)
    # @method_decorator(permission_required('api.add_roles', raise_exception=True))
    def create(self, request):
        try:
            return super().create(request)
        except Exception as e:
            print("Error during creation:", e)
            return Response({"detail": str(e)}, status=status.HTTP_400_BAD_REQUEST)

    def get_queryset(self):
        return super().get_queryset()

    @method_decorator(login_required)
    @method_decorator(permission_required("api.change_roles", raise_exception=True))
    def update(self, request, pk=None):
        return super().update(request, pk)

    @method_decorator(login_required)
    @method_decorator(permission_required("api.delete_roles", raise_exception=True))
    def destroy(self, request, pk=None):
        return super().destroy(request, pk)
