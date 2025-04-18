"""The model, serializer, and viewset for Actor."""

from django.db import models
from django.contrib.auth.decorators import permission_required, login_required
from django.utils.decorators import method_decorator

from rest_framework.serializers import ModelSerializer
from rest_framework.viewsets import ModelViewSet
from django_filters.rest_framework import DjangoFilterBackend


class Actor(models.Model):
    """ """

    originated_by = models.TextField(default="", blank=True)
    originated_date = models.DateTimeField(auto_now_add=True)
    last_updated_by = models.TextField(default="", blank=True)
    last_updated_date = models.DateTimeField(auto_now=True)

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "Actor"


class ActorSerializer(ModelSerializer):
    """The serializer."""

    class Meta:
        """Settings for the serializer."""

        model = Actor
        fields = "__all__"


class ActorViewSet(ModelViewSet):
    """The viewset for the model."""

    queryset = Actor.objects.order_by("pk")
    serializer_class = ActorSerializer
    filter_backends = [DjangoFilterBackend]
    filterset_fields = [""]

    @method_decorator(login_required)
    @method_decorator(permission_required("api.add_actor", raise_exception=True))
    def create(self, request):
        return super().create(request)

    def get_queryset(self):
        return super().get_queryset()

    @method_decorator(login_required)
    @method_decorator(permission_required("api.change_actor", raise_exception=True))
    def update(self, request, pk=None):
        return super().update(request, pk)

    @method_decorator(login_required)
    @method_decorator(permission_required("api.delete_actor", raise_exception=True))
    def destroy(self, request, pk=None):
        return super().destroy(request, pk)
