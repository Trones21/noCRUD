from django.db import models
from rest_framework import serializers, viewsets
from .character import Character
from django.contrib.auth import get_user_model

User = get_user_model()


class BackstoryPitch(models.Model):
    character = models.ForeignKey(
        Character, on_delete=models.CASCADE, related_name="pitches"
    )
    user = models.ForeignKey(User, on_delete=models.CASCADE)
    pitch_text = models.TextField()
    upvotes = models.IntegerField(default=0)
    visibility = models.CharField(max_length=20, default="public")

    def __str__(self):
        return f"Pitch for {self.character.name}"

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "backstory"


class BackstoryPitchSerializer(serializers.ModelSerializer):
    class Meta:
        model = BackstoryPitch
        fields = "__all__"


class BackstoryPitchViewSet(viewsets.ModelViewSet):
    queryset = BackstoryPitch.objects.all()
    serializer_class = BackstoryPitchSerializer
