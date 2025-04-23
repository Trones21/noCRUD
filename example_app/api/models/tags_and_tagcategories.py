from django.db import models
from django.contrib.auth.decorators import permission_required, login_required
from django.utils.decorators import method_decorator

from rest_framework.serializers import ModelSerializer
from rest_framework.viewsets import ModelViewSet
from django_filters.rest_framework import DjangoFilterBackend


class TagCategory(models.Model):
    name = models.CharField(max_length=255)

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "tag_category"


class TagCategorySerializer(ModelSerializer):
    """The serializer."""

    class Meta:
        """Settings for the serializer."""

        model = TagCategory
        fields = "__all__"


class TagCategoryViewSet(ModelViewSet):
    """The viewset for the model."""

    queryset = TagCategory.objects.order_by("pk")
    serializer_class = TagCategorySerializer
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ["name"]

    @method_decorator(login_required)
    @method_decorator(permission_required("api.add_tagcategory", raise_exception=True))
    def create(self, request):
        return super().create(request)

    def get_queryset(self):
        print("self", self)
        return super().get_queryset()


class Tag(models.Model):
    name = models.CharField(max_length=50)
    category = models.ForeignKey(TagCategory, on_delete=models.PROTECT)

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "tag"


class TagSerializer(ModelSerializer):
    """The serializer."""

    class Meta:
        """Settings for the serializer."""

        model = Tag
        fields = "__all__"


class TagViewSet(ModelViewSet):
    """The viewset for the model."""

    queryset = Tag.objects.order_by("pk")
    serializer_class = TagSerializer
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ["name", "category"]

    @method_decorator(login_required)
    @method_decorator(permission_required("api.add_Tag", raise_exception=True))
    def create(self, request):
        return super().create(request)

    def get_queryset(self):
        return super().get_queryset()

    @method_decorator(login_required)
    @method_decorator(permission_required("api.change_Tag", raise_exception=True))
    def update(self, request, pk=None):
        return super().update(request, pk)

    @method_decorator(login_required)
    @method_decorator(permission_required("api.delete_Tag", raise_exception=True))
    def destroy(self, request, pk=None):
        return super().destroy(request, pk)
