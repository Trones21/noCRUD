from django.db import models
from rest_framework import serializers, viewsets
from .production import Production


class Character(models.Model):
    name = models.CharField(max_length=255)
    actor = models.CharField(max_length=255, blank=True)
    description = models.TextField(blank=True)
    production = models.ForeignKey(
        Production, on_delete=models.CASCADE, related_name="characters"
    )

    def __str__(self):
        return self.name

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "character"


class CharacterSerializer(serializers.ModelSerializer):
    class Meta:
        model = Character
        fields = "__all__"


class CharacterViewSet(viewsets.ModelViewSet):
    queryset = Character.objects.all()
    serializer_class = CharacterSerializer
