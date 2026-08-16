from django.db import models
from django.conf import settings
from rest_framework import serializers, viewsets, status
from rest_framework.response import Response
from rest_framework.decorators import action
from .roles import Roles, RolesSerializer
from rest_framework.request import Request as DRFRequest


class UserRole(models.Model):
    user = models.ForeignKey(
        settings.AUTH_USER_MODEL, on_delete=models.CASCADE, related_name="user_roles"
    )
    role = models.ForeignKey(
        "api.Roles", on_delete=models.CASCADE, related_name="user_roles"
    )

    class Meta:
        db_table = "users_roles"
        constraints = [
            models.UniqueConstraint(
                fields=["user", "role"], name="unique_user_role_field"
            )
        ]


class UserRoleSerializer(serializers.ModelSerializer):
    class Meta:
        model = UserRole
        fields = ["user", "role"]


class UserRoleViewSet(viewsets.ModelViewSet):
    request: DRFRequest
    queryset = UserRole.objects.all()  # Default queryset
    serializer_class = UserRoleSerializer  # Default serializer

    def get_serializer_class(self):
        if self.action == "list":
            returnRoles = (
                self.request.query_params.get("returnRoles", "false").lower() == "true"
            )
            if returnRoles:
                return RolesSerializer
        return UserRoleSerializer

    def get_queryset(self):
        if self.action == "list":
            returnRoles = (
                self.request.query_params.get("returnRoles", "false").lower() == "true"
            )
            if returnRoles:
                user_id = self.request.query_params.get("user_id", 1)
                user_roles = UserRole.objects.filter(user_id=user_id)
                role_ids = user_roles.values_list("role_id", flat=True)
                return Roles.objects.filter(role_id__in=role_ids)
        return super().get_queryset()

    def list(self, request, *args, **kwargs):
        queryset = self.get_queryset()
        serializer = self.get_serializer(queryset, many=True)
        return Response(serializer.data, status=status.HTTP_200_OK)

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        self.perform_create(serializer)
        return Response(serializer.data, status=status.HTTP_201_CREATED)

    def perform_create(self, serializer):
        serializer.save()

    @action(methods=["delete"], detail=False)
    def delete(self, request):
        user_id = request.data.get("user_id")
        role_id = request.data.get("role_id")

        if not user_id or not role_id:
            return Response(
                {"error": "user_id and role_id are required."},
                status=status.HTTP_400_BAD_REQUEST,
            )

        try:
            user_role = UserRole.objects.get(user_id=user_id, role_id=role_id)
            user_role.delete()
            return Response(status=status.HTTP_204_NO_CONTENT)
        except UserRole.DoesNotExist:
            return Response(
                {"error": "UserRole not found."}, status=status.HTTP_404_NOT_FOUND
            )
