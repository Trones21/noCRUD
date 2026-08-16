from django.db import models
from rest_framework import serializers, viewsets

from api.models.actor import Actor
from .production import Production


class Character(models.Model):
    name = models.CharField(max_length=255)
    actor = models.ForeignKey(
        Actor,
        null=True,
        blank=True,
        on_delete=models.SET_NULL,
        related_name="characters",
    )
    description = models.TextField(blank=True)
    productions = models.ManyToManyField(
        Production,
        related_name="characters",
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
