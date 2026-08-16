from django.db import models
from rest_framework import serializers, viewsets


class Universe(models.Model):
    name = models.CharField(max_length=255)
    description = models.TextField(blank=True)

    def __str__(self):
        return self.name

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "universe"


class UniverseSerializer(serializers.ModelSerializer):
    class Meta:
        model = Universe
        fields = "__all__"


class UniverseViewSet(viewsets.ModelViewSet):
    queryset = Universe.objects.all()
    serializer_class = UniverseSerializer
